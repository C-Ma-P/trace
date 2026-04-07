package phoneintake

import (
	"time"

	"componentmanager/internal/sourcing"
)

// IntakeEvent records a recent phone intake action for desktop visibility.
type IntakeEvent struct {
	Timestamp   time.Time `json:"timestamp"`
	QRData      string    `json:"qrData"`
	ComponentID string    `json:"componentId,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
	Action      string    `json:"action"` // "lookup" or "submit"
	NewQuantity *int      `json:"newQuantity,omitempty"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
}

// ScanRequest is sent by the phone when a barcode is detected and routed.
type ScanRequest struct {
	Vendor   string `json:"vendor"`   // "LCSC" or "Mouser"
	Format   string `json:"format"`   // barcode format, e.g. "qr_code", "data_matrix"
	RawValue string `json:"rawValue"` // raw decoded string from the detector
}

// ResolvedComponent holds the resolved component data for a pending scan.
type ResolvedComponent struct {
	ComponentID  string `json:"componentId"`
	MPN          string `json:"mpn"`
	Manufacturer string `json:"manufacturer"`
	Package      string `json:"package"`
	Description  string `json:"description"`
	ImageURL     string `json:"imageUrl"`
	ProductURL   string `json:"productUrl"`
}

// ScanResponse is returned after processing a scan.
type ScanResponse struct {
	OK           bool               `json:"ok"`
	Error        string             `json:"error,omitempty"`
	ID           string             `json:"id,omitempty"`
	Quantity     string             `json:"quantity,omitempty"`
	Resolved     *ResolvedComponent `json:"resolved,omitempty"`
	ResolveError string             `json:"resolveError,omitempty"`
}

// PendingScan holds a decoded scan in memory awaiting user confirmation.
type PendingScan struct {
	ID        string             `json:"id"`
	Timestamp time.Time          `json:"timestamp"`
	Vendor    string             `json:"vendor"`
	Format    string             `json:"format"`
	RawValue  string             `json:"rawValue"`
	Resolved  *ResolvedComponent `json:"resolved,omitempty"`
	Error     string             `json:"error,omitempty"`
	Quantity  string             `json:"quantity"`
	// Offer holds the raw supplier data; not exposed to the frontend.
	// Component creation is deferred until the user confirms the scan.
	Offer *sourcing.SupplierOffer `json:"-"`
}

// DetailResponse is returned by the /api/detail endpoint.
type DetailResponse struct {
	OK    bool        `json:"ok"`
	Error string      `json:"error,omitempty"`
	Scan  PendingScan `json:"scan,omitempty"`
}

// ConfirmRequest is sent by the phone to register a pending scan.
type ConfirmRequest struct {
	ID       string `json:"id"`
	Quantity int    `json:"quantity"`
}

// ConfirmResponse is returned after confirming a scan.
type ConfirmResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// StatusInfo is returned to the desktop UI.
type StatusInfo struct {
	URL    string        `json:"url"`
	Port   int           `json:"port"`
	Recent []IntakeEvent `json:"recent"`
}
