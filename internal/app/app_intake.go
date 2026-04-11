package app

import (
	"context"
	"fmt"

	"trace/internal/domain"
	"trace/internal/phoneintake"
)

// SetIntakeServer sets the phone intake server reference for status queries.
func (a *App) SetIntakeServer(s *phoneintake.Server) {
	a.intake = s
}

// SetBagRepo sets the inventory bag repository.
func (a *App) SetBagRepo(r domain.InventoryBagRepository) {
	a.bagRepo = r
}

// ---------- Phone Intake Status ----------

// PhoneIntakeHostInfo exposes the result of LAN host detection to the frontend.
type PhoneIntakeHostInfo struct {
	Host   string `json:"host"`
	Iface  string `json:"iface"`
	Source string `json:"source"` // "auto", "override", or "fallback"
}

type PhoneIntakeInfoResponse struct {
	Available bool                      `json:"available"`
	Active    bool                      `json:"active"`
	URL       string                    `json:"url"`
	Port      int                       `json:"port"`
	Recent    []phoneintake.IntakeEvent `json:"recent"`
	HostInfo  PhoneIntakeHostInfo       `json:"hostInfo"`
}

func (a *App) GetPhoneIntakeInfo() PhoneIntakeInfoResponse {
	if a.intake == nil {
		return PhoneIntakeInfoResponse{}
	}
	sel := a.intake.HostInfo()
	return PhoneIntakeInfoResponse{
		Available: true,
		Active:    a.intake.IsRunning(),
		URL:       a.intake.PhoneURL(),
		Port:      a.intake.Port(),
		Recent:    a.intake.RecentEvents(),
		HostInfo: PhoneIntakeHostInfo{
			Host:   sel.Host,
			Iface:  sel.Iface,
			Source: sel.Source,
		},
	}
}

func (a *App) SetPhoneIntakeEnabled(enabled bool) error {
	if a.intake == nil {
		return fmt.Errorf("phone intake not available")
	}
	if enabled {
		return a.intake.Start()
	}
	a.intake.Stop()
	return nil
}

// SetPhoneIntakeHostOverride sets a manual display-host override for the phone
// intake URL. The change is persisted and takes effect immediately in
// PhoneURL(); restarting the server regenerates the PKI cert with the new SAN.
func (a *App) SetPhoneIntakeHostOverride(host string) error {
	if a.intake == nil {
		return fmt.Errorf("phone intake not available")
	}
	return a.intake.SetDisplayHostOverride(host)
}

// ClearPhoneIntakeHostOverride removes the manual display-host override and
// reverts to auto-detection.
func (a *App) ClearPhoneIntakeHostOverride() error {
	if a.intake == nil {
		return fmt.Errorf("phone intake not available")
	}
	return a.intake.ClearDisplayHostOverride()
}

// StopIntakeIfRunning stops the phone intake server if it is currently running.
// Called when the last project window closes so the server doesn't run silently
// while only the launcher is visible.
func (a *App) StopIntakeIfRunning() {
	if a.intake != nil {
		a.intake.Stop()
	}
}

// ---------- Inventory Bags ----------

type CreateBagInput struct {
	ComponentID string `json:"componentId"`
	Label       string `json:"label"`
	QRData      string `json:"qrData"`
}

type BagResponse struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	QRData      string `json:"qrData"`
	ComponentID string `json:"componentId"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

func (a *App) CreateInventoryBag(input CreateBagInput) (BagResponse, error) {
	if err := a.checkReady(); err != nil {
		return BagResponse{}, err
	}
	if a.bagRepo == nil {
		return BagResponse{}, errNoBagRepo
	}
	bag := domain.InventoryBag{
		ID:          newID(),
		Label:       input.Label,
		QRData:      input.QRData,
		ComponentID: input.ComponentID,
	}
	bag, err := a.bagRepo.CreateBag(context.Background(), bag)
	if err != nil {
		return BagResponse{}, err
	}
	return bagToResponse(bag), nil
}

func (a *App) ListComponentBags(componentID string) ([]BagResponse, error) {
	if err := a.checkReady(); err != nil {
		return nil, err
	}
	if a.bagRepo == nil {
		return nil, errNoBagRepo
	}
	bags, err := a.bagRepo.ListBagsByComponent(context.Background(), componentID)
	if err != nil {
		return nil, err
	}
	out := make([]BagResponse, len(bags))
	for i, b := range bags {
		out[i] = bagToResponse(b)
	}
	return out, nil
}

func (a *App) DeleteInventoryBag(id string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	if a.bagRepo == nil {
		return errNoBagRepo
	}
	return a.bagRepo.DeleteBag(context.Background(), id)
}

func bagToResponse(b domain.InventoryBag) BagResponse {
	return BagResponse{
		ID:          b.ID,
		Label:       b.Label,
		QRData:      b.QRData,
		ComponentID: b.ComponentID,
		CreatedAt:   b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   b.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

var errNoBagRepo = fmt.Errorf("inventory bag repository not available")
