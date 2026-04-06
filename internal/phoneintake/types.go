package phoneintake

import "time"

// LookupRequest is sent by the phone to resolve a scanned QR code.
type LookupRequest struct {
	QRData string `json:"qrData"`
}

// LookupResponse contains resolved component information or an unresolved state.
type LookupResponse struct {
	Found        bool   `json:"found"`
	ComponentID  string `json:"componentId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	Manufacturer string `json:"manufacturer,omitempty"`
	MPN          string `json:"mpn,omitempty"`
	Description  string `json:"description,omitempty"`
	Package      string `json:"package,omitempty"`
	Quantity     *int   `json:"quantity"`
	QuantityMode string `json:"quantityMode,omitempty"`
	Location     string `json:"location,omitempty"`
	ImageURL     string `json:"imageUrl,omitempty"`
	BagLabel     string `json:"bagLabel,omitempty"`
	RawQR        string `json:"rawQr"`
}

// SubmitRequest is sent by the phone to update inventory.
type SubmitRequest struct {
	ComponentID string `json:"componentId"`
	Mode        string `json:"mode"` // "set" or "delta"
	Value       int    `json:"value"`
}

// SubmitResponse confirms the inventory update.
type SubmitResponse struct {
	Success     bool   `json:"success"`
	Quantity    *int   `json:"quantity"`
	DisplayName string `json:"displayName"`
	Error       string `json:"error,omitempty"`
}

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

// StatusInfo is returned to the desktop UI.
type StatusInfo struct {
	URL    string        `json:"url"`
	Port   int           `json:"port"`
	Recent []IntakeEvent `json:"recent"`
}
