package phoneintake

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"

	"sync"
	"time"

	"trace/internal/activity"
	"trace/internal/domain"
	"trace/internal/service"
	"trace/internal/sourcing"
)

const (
	defaultPort     = 8741
	maxRecentEvents = 50
	maxPendingScans = 500
	stableHostname  = "trace.local"
)

// Server runs a local HTTP server for phone-based inventory intake.
type Server struct {
	svc       *service.Service
	bags      domain.InventoryBagRepository
	comps     domain.ComponentRepository
	emitter   activity.Emitter
	token     string
	port      int
	pkiDir    string
	mu        sync.Mutex
	running   bool
	recent    []IntakeEvent
	pending   map[string]*PendingScan
	httpSrv   *http.Server
	lanIP     string
	mdnsStop  func()
	caCertPEM []byte
}

func NewServer(
	svc *service.Service,
	comps domain.ComponentRepository,
	bags domain.InventoryBagRepository,
	port int,
	emitter activity.Emitter,
	pkiDir string,
) *Server {
	if port == 0 {
		port = defaultPort
	}
	if emitter == nil {
		emitter = activity.NopEmitter
	}
	token := generateToken()
	return &Server{
		svc:      svc,
		bags:     bags,
		comps:    comps,
		emitter:  emitter,
		token:    token,
		port:     port,
		pkiDir:   pkiDir,
		lanIP:    detectLANIP(),
		pending:  make(map[string]*PendingScan),
		mdnsStop: func() {},
	}
}

func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	pki, err := LoadOrCreatePKI(s.pkiDir, s.lanIP)
	if err != nil {
		return fmt.Errorf("phone-intake: load PKI: %w", err)
	}
	s.caCertPEM = pki.CACertPEM

	s.mdnsStop = startMDNS(s.lanIP, s.emitter)

	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", s.port), pki.TLSConfig)
	if err != nil {
		return fmt.Errorf("phone-intake: bind port %d: %w", s.port, err)
	}

	mux := http.NewServeMux()
	prefix := "/phone/" + s.token
	mux.HandleFunc(prefix, s.handlePage)
	mux.HandleFunc(prefix+"/ca.crt", s.handleCACert)
	mux.HandleFunc(prefix+"/api/recent", s.handleRecent)
	mux.HandleFunc(prefix+"/api/scan", s.handleScan)
	mux.HandleFunc(prefix+"/api/detail", s.handleDetail)
	mux.HandleFunc(prefix+"/api/confirm", s.handleConfirm)

	s.httpSrv = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	s.running = true
	s.emitter.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "server-started", "Phone intake server started", map[string]any{"port": s.port, "url": s.PhoneURL()}))

	go func() {
		log.Printf("[phone-intake] serving on :%d → %s", s.port, s.PhoneURL())
		if err := s.httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[phone-intake] server error: %v", err)
		}
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	return nil
}

func (s *Server) Stop() {
	s.mu.Lock()
	srv := s.httpSrv
	stopMDNS := s.mdnsStop
	wasRunning := s.running
	s.running = false
	s.mdnsStop = func() {}
	s.mu.Unlock()

	if wasRunning {
		s.emitter.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "server-stopped", "Phone intake server stopped", map[string]any{"port": s.port}))
	}

	stopMDNS()

	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}
}

// IsRunning reports whether the server is currently listening.
func (s *Server) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.running
}

func (s *Server) PhoneURL() string {
	return fmt.Sprintf("https://%s:%d/phone/%s", stableHostname, s.port, s.token)
}

func (s *Server) CACertPEM() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.caCertPEM
}

// Port returns the configured port.
func (s *Server) Port() int { return s.port }

// RecentEvents returns a copy of recent intake events (newest first).
func (s *Server) RecentEvents() []IntakeEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]IntakeEvent, len(s.recent))
	copy(out, s.recent)
	return out
}

func (s *Server) addEvent(ev IntakeEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recent = append([]IntakeEvent{ev}, s.recent...)
	if len(s.recent) > maxRecentEvents {
		s.recent = s.recent[:maxRecentEvents]
	}
}

// ---------- handlers ----------

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(phonePage(s.token)))
}

func (s *Server) handleCACert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	cert := s.caCertPEM
	s.mu.Unlock()
	if len(cert) == 0 {
		http.Error(w, "certificate not available", http.StatusServiceUnavailable)
		return
	}
	s.emitter.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "ca-cert-served", "CA certificate downloaded by phone", map[string]any{"remoteAddr": r.RemoteAddr}))
	w.Header().Set("Content-Type", "application/x-x509-ca-cert")
	w.Header().Set("Content-Disposition", `attachment; filename="trace-ca.crt"`)
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(cert)
}

func (s *Server) handleRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.RecentEvents())
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		s.handleScanPost(w, r)
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleScanPost(w http.ResponseWriter, r *http.Request) {
	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ScanResponse{Error: "invalid request"})
		return
	}

	var vendorPartID, quantity string
	switch req.Vendor {
	case sourcing.ProviderLCSC:
		vendorPartID, quantity = parseLcscBarcode(req.RawValue)
		log.Printf("[phone-intake] LCSC scan: partID=%s qty=%s", vendorPartID, quantity)
	case sourcing.ProviderMouser:
		vendorPartID, quantity = parseMouserBarcode(req.RawValue)
		log.Printf("[phone-intake] Mouser scan: part=%s qty=%s", vendorPartID, quantity)
	default:
		log.Printf("[phone-intake] unknown vendor %q format=%s", req.Vendor, req.Format)
		writeJSON(w, http.StatusOK, ScanResponse{OK: true})
		return
	}

	id := generateToken()
	scan := &PendingScan{
		ID:        id,
		Timestamp: time.Now(),
		Vendor:    req.Vendor,
		Format:    req.Format,
		RawValue:  req.RawValue,
		Quantity:  quantity,
	}

	resp := ScanResponse{OK: true, ID: id, Quantity: quantity}

	s.emitter.Emit(activity.NewPhoneEvent(activity.SeverityInfo, "scan-received", "Phone scan received", map[string]any{"vendor": req.Vendor, "format": req.Format, "rawValue": req.RawValue, "quantity": quantity}))

	if vendorPartID != "" {
		offer, err := s.svc.LookupVendorPartID(r.Context(), req.Vendor, vendorPartID)
		if err != nil {
			scan.Error = err.Error()
			resp.ResolveError = err.Error()
			s.emitter.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "lookup-failed", "Vendor lookup failed", map[string]any{"vendor": req.Vendor, "partId": vendorPartID, "error": err.Error()}))
			log.Printf("[phone-intake] lookup failed vendor=%s partID=%s: %v", req.Vendor, vendorPartID, err)
		} else {
			// Populate display data from the offer without touching the DB.
			// Component creation is deferred until handleConfirm.
			resolved := &ResolvedComponent{
				MPN:             offer.MPN,
				Manufacturer:    offer.Manufacturer,
				Package:         offer.Package,
				Description:     offer.Description,
				ImageURL:        offer.ImageURL,
				ProductURL:      offer.ProductURL,
				HasSymbol:       offer.HasSymbol,
				HasFootprint:    offer.HasFootprint,
				HasDatasheet:    offer.HasDatasheet,
				AssetProbeState: string(offer.AssetProbeState),
				AssetProbeError: offer.AssetProbeError,
			}
			scan.Resolved = resolved
			scan.Offer = &offer
			resp.Resolved = resolved
			s.emitter.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "lookup-succeeded", "Vendor lookup succeeded", map[string]any{"vendor": req.Vendor, "partId": vendorPartID, "mpn": offer.MPN}))
		}
	} else {
		scan.Error = "no part ID found in barcode"
		resp.ResolveError = scan.Error
		s.emitter.Emit(activity.NewPhoneEvent(activity.SeverityWarning, "scan-parse-failed", "Phone scan parse failed", map[string]any{"vendor": req.Vendor, "format": req.Format, "rawValue": req.RawValue}))
	}

	s.mu.Lock()
	s.pending[id] = scan
	if len(s.pending) > maxPendingScans {
		var oldestID string
		var oldestTime time.Time
		for pid, ps := range s.pending {
			if oldestID == "" || ps.Timestamp.Before(oldestTime) {
				oldestID = pid
				oldestTime = ps.Timestamp
			}
		}
		delete(s.pending, oldestID)
	}
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, DetailResponse{Error: "missing id"})
		return
	}
	s.mu.Lock()
	scan, ok := s.pending[id]
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, DetailResponse{Error: "not found"})
		return
	}
	writeJSON(w, http.StatusOK, DetailResponse{OK: true, Scan: *scan})
}

func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req ConfirmRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, ConfirmResponse{Error: "invalid request"})
		return
	}
	s.mu.Lock()
	scan, ok := s.pending[req.ID]
	if ok {
		delete(s.pending, req.ID)
	}
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusNotFound, ConfirmResponse{Error: "scan not found"})
		return
	}
	if scan.Offer == nil {
		writeJSON(w, http.StatusBadRequest, ConfirmResponse{Error: "component not resolved"})
		return
	}
	component, err := s.svc.ResolveComponentFromOffer(r.Context(), *scan.Offer)
	if err != nil {
		log.Printf("[phone-intake] resolve failed on confirm scan=%s: %v", req.ID, err)
		writeJSON(w, http.StatusInternalServerError, ConfirmResponse{Error: err.Error()})
		return
	}
	if req.Quantity > 0 {
		if _, err := s.svc.StampInventory(r.Context(), component.ID, req.Quantity); err != nil {
			log.Printf("[phone-intake] inventory stamp failed componentID=%s qty=%d: %v", component.ID, req.Quantity, err)
			writeJSON(w, http.StatusInternalServerError, ConfirmResponse{Error: err.Error()})
			return
		}
	}
	s.emitter.Emit(activity.NewPhoneEvent(activity.SeveritySuccess, "scan-imported", "Phone scan imported", map[string]any{"scanId": req.ID, "vendor": scan.Vendor, "componentId": component.ID, "quantity": req.Quantity}))
	log.Printf("[phone-intake] confirmed scan %s vendor=%s componentID=%s qty=%d", req.ID, scan.Vendor, component.ID, req.Quantity)
	writeJSON(w, http.StatusOK, ConfirmResponse{OK: true})
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func generateToken() string {
	buf := make([]byte, 12)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func detectLANIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "localhost"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "localhost"
}
