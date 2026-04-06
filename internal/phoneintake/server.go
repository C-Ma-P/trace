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
	"strings"
	"sync"
	"time"

	"componentmanager/internal/domain"
	"componentmanager/internal/service"
)

const (
	defaultPort     = 8741
	maxRecentEvents = 50
)

// Server runs a local HTTP server for phone-based inventory intake.
type Server struct {
	svc     *service.Service
	bags    domain.InventoryBagRepository
	comps   domain.ComponentRepository
	token   string
	port    int
	mu      sync.Mutex
	running bool
	recent  []IntakeEvent
	httpSrv *http.Server
	lanIP   string
}

// NewServer creates a phone intake server. Port 0 uses the default (8741).
func NewServer(
	svc *service.Service,
	comps domain.ComponentRepository,
	bags domain.InventoryBagRepository,
	port int,
) *Server {
	if port == 0 {
		port = defaultPort
	}
	token := generateToken()
	return &Server{
		svc:   svc,
		bags:  bags,
		comps: comps,
		token: token,
		port:  port,
		lanIP: detectLANIP(),
	}
}

// Start binds the port and begins serving in a background goroutine.
// Returns an error immediately if the port cannot be bound.
// Safe to call again after Stop.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil
	}

	tlsCfg, err := selfSignedTLS(s.lanIP)
	if err != nil {
		return fmt.Errorf("phone-intake: generate TLS cert: %w", err)
	}

	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", s.port), tlsCfg)
	if err != nil {
		return fmt.Errorf("phone-intake: bind port %d: %w", s.port, err)
	}

	mux := http.NewServeMux()
	prefix := "/phone/" + s.token
	mux.HandleFunc(prefix, s.handlePage)
	mux.HandleFunc(prefix+"/api/lookup", s.handleLookup)
	mux.HandleFunc(prefix+"/api/submit", s.handleSubmit)
	mux.HandleFunc(prefix+"/api/recent", s.handleRecent)

	s.httpSrv = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	s.running = true

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

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	s.mu.Lock()
	srv := s.httpSrv
	s.running = false
	s.mu.Unlock()

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

// PhoneURL returns the full URL a phone should open.
func (s *Server) PhoneURL() string {
	host := s.lanIP
	if host == "" {
		host = "localhost"
	}
	return fmt.Sprintf("https://%s:%d/phone/%s", host, s.port, s.token)
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

func (s *Server) handleLookup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req LookupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, LookupResponse{RawQR: ""})
		return
	}

	qr := strings.TrimSpace(req.QRData)
	if qr == "" {
		writeJSON(w, http.StatusBadRequest, LookupResponse{RawQR: ""})
		return
	}

	ctx := context.Background()
	resp := s.lookupQR(ctx, qr)

	action := "lookup"
	ev := IntakeEvent{
		Timestamp:   time.Now(),
		QRData:      qr,
		ComponentID: resp.ComponentID,
		DisplayName: resp.DisplayName,
		Action:      action,
		Success:     resp.Found,
	}
	s.addEvent(ev)

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, SubmitResponse{Error: "invalid request"})
		return
	}

	if req.ComponentID == "" {
		writeJSON(w, http.StatusBadRequest, SubmitResponse{Error: "componentId required"})
		return
	}
	if req.Mode != "set" && req.Mode != "delta" {
		writeJSON(w, http.StatusBadRequest, SubmitResponse{Error: "mode must be 'set' or 'delta'"})
		return
	}

	ctx := context.Background()
	var comp domain.Component
	var err error

	if req.Mode == "delta" {
		comp, err = s.svc.AdjustComponentQuantity(ctx, req.ComponentID, req.Value)
	} else {
		// Set exact quantity
		existing, getErr := s.comps.GetComponent(ctx, req.ComponentID)
		if getErr != nil {
			writeJSON(w, http.StatusNotFound, SubmitResponse{Error: "component not found"})
			return
		}
		qty := req.Value
		existing.Quantity = &qty
		if existing.QuantityMode == domain.QuantityModeUnknown || existing.QuantityMode == "" {
			existing.QuantityMode = domain.QuantityModeExact
		}
		comp, err = s.svc.UpdateComponentInventory(ctx, existing)
	}

	displayName := componentDisplayName(comp)
	ev := IntakeEvent{
		Timestamp:   time.Now(),
		ComponentID: req.ComponentID,
		DisplayName: displayName,
		Action:      "submit",
	}

	if err != nil {
		ev.Success = false
		ev.Error = err.Error()
		s.addEvent(ev)
		writeJSON(w, http.StatusInternalServerError, SubmitResponse{Error: err.Error()})
		return
	}

	ev.Success = true
	ev.NewQuantity = comp.Quantity
	s.addEvent(ev)

	writeJSON(w, http.StatusOK, SubmitResponse{
		Success:     true,
		Quantity:    comp.Quantity,
		DisplayName: displayName,
	})
}

func (s *Server) handleRecent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.RecentEvents())
}

// ---------- lookup logic ----------

func (s *Server) lookupQR(ctx context.Context, qr string) LookupResponse {
	// 1. Try bag lookup by exact QR data
	bag, err := s.bags.GetBagByQRData(ctx, qr)
	if err == nil {
		comp, compErr := s.comps.GetComponent(ctx, bag.ComponentID)
		if compErr == nil {
			return s.componentToLookup(comp, bag.Label, qr)
		}
	}

	// 2. Fallback: try matching by MPN (case-insensitive)
	comps, err := s.comps.FindComponents(ctx, domain.ComponentFilter{MPN: qr})
	if err == nil && len(comps) == 1 {
		return s.componentToLookup(comps[0], "", qr)
	}

	// 3. Unresolved
	return LookupResponse{Found: false, RawQR: qr}
}

func (s *Server) componentToLookup(c domain.Component, bagLabel, rawQR string) LookupResponse {
	imageURL := s.bags.FindComponentImageURL(context.Background(), c.ID)
	return LookupResponse{
		Found:        true,
		ComponentID:  c.ID,
		DisplayName:  componentDisplayName(c),
		Manufacturer: c.Manufacturer,
		MPN:          c.MPN,
		Description:  c.Description,
		Package:      c.Package,
		Quantity:     c.Quantity,
		QuantityMode: string(c.QuantityMode),
		Location:     c.Location,
		ImageURL:     imageURL,
		BagLabel:     bagLabel,
		RawQR:        rawQR,
	}
}

// ---------- helpers ----------

func componentDisplayName(c domain.Component) string {
	if c.MPN != "" && c.Manufacturer != "" {
		return c.Manufacturer + " " + c.MPN
	}
	if c.MPN != "" {
		return c.MPN
	}
	if c.Description != "" {
		return c.Description
	}
	return c.ID
}

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
