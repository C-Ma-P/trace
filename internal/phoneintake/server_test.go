package phoneintake

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/C-Ma-P/trace/internal/activity"
	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
	"github.com/C-Ma-P/trace/internal/service"
	"github.com/C-Ma-P/trace/internal/sourcing"
)

type phoneStubProvider struct {
	name   string
	offers map[string]sourcing.SupplierOffer
	last   struct {
		vendor       string
		partID       string
		manufacturer string
	}
}

func (p *phoneStubProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "phone-stub"
}

func (p *phoneStubProvider) Enabled() bool {
	return true
}

func (p *phoneStubProvider) Search(_ context.Context, _ sourcing.RequirementQuery) ([]sourcing.SupplierOffer, error) {
	return nil, nil
}

func (p *phoneStubProvider) FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (p *phoneStubProvider) LookupByVendorPartID(_ context.Context, vendor, partID string) (sourcing.SupplierOffer, error) {
	p.last.vendor = vendor
	p.last.partID = partID
	offer, ok := p.offers[vendor+":"+partID]
	if !ok {
		return sourcing.SupplierOffer{}, domain.ErrNotFound{ID: partID}
	}
	return offer, nil
}

func (p *phoneStubProvider) LookupByPartNumberAndManufacturer(_ context.Context, partID, manufacturer string) (sourcing.SupplierOffer, error) {
	p.last.vendor = sourcing.ProviderMouser
	p.last.partID = partID
	p.last.manufacturer = manufacturer
	if offer, ok := p.offers[sourcing.ProviderMouser+":"+manufacturer+":"+partID]; ok {
		return offer, nil
	}
	return p.LookupByVendorPartID(context.Background(), sourcing.ProviderMouser, partID)
}

func TestPhoneIntake_LCSCScanConfirm_CreatesComponentAttrsAndQuantity(t *testing.T) {
	runPhoneConfirmRegression(t,
		sourcing.ProviderLCSC,
		"qr_code",
		"{pc:C25804,qty:1000}",
		"C25804",
		1000,
		sourcing.SupplierOffer{
			Provider:     sourcing.ProviderLCSC,
			Manufacturer: "Yageo",
			MPN:          "RC0402FR-0710KL",
			Package:      "Reel",
			Description:  "Chip Resistor 10 kOhms ±1% 0402 100mW",
			Raw: map[string]string{
				"catalog":       "Chip Resistor - Surface Mount",
				"Resistance":    "10 kOhms",
				"Tolerance":     "1%",
				"Power":         "100mW",
				"EncapStandard": "0402",
			},
		},
	)
}

func TestPhoneIntake_MouserScanConfirm_CreatesComponentAttrsAndQuantity(t *testing.T) {
	runPhoneConfirmRegression(t,
		sourcing.ProviderMouser,
		"qr_code",
		"21P603-RC0402FR-0710KLQ2500",
		"603-RC0402FR-0710KL",
		2500,
		sourcing.SupplierOffer{
			Provider:     sourcing.ProviderMouser,
			Manufacturer: "Yageo",
			MPN:          "RC0402FR-0710KL",
			Package:      "Cut Tape",
			Description:  "Chip Resistor 10,000 Ohms 1% 0402 1/10W",
			Raw: map[string]string{
				"category":             "Chip Resistors",
				"Ohms":                 "10,000 Ohms",
				"Resistance Tolerance": "1%",
				"Power (Watts)":        "1/10W",
				"Case Code - in":       "0402",
			},
		},
	)
}

func TestPhoneIntake_MouserDataMatrixConfirm_UsesManufacturerLookupForRichAttrs(t *testing.T) {
	provider := &phoneStubProvider{name: sourcing.ProviderMouser, offers: map[string]sourcing.SupplierOffer{
		sourcing.ProviderMouser + ":0603SAF1004TCE": {
			Provider:     sourcing.ProviderMouser,
			Manufacturer: "Royalohm",
			MPN:          "0603SAF1004TCE",
			Package:      "Reel",
			Description:  "Thick Film Resistors - SMD RMC 0603 1/10W-S 1% T/R-10000",
		},
		sourcing.ProviderMouser + ":Royalohm:0603SAF1004TCE": {
			Provider:     sourcing.ProviderMouser,
			Manufacturer: "Royalohm",
			MPN:          "0603SAF1004TCE",
			Package:      "Reel",
			Description:  "Thick Film Resistors - SMD RMC 0603 1/10W-S 1% T/R-10000",
			Raw: map[string]string{
				"category":             "Thick Film Resistors - SMD",
				"Resistance":           "1 MOhms",
				"Resistance Tolerance": "1%",
				"Power (Watts)":        "1/10W",
				"TCR (ppm/C)":          "100 ppm/C",
				"Case Code - in":       "0603",
				"Technology":           "Thick Film",
			},
		},
	}}
	compRepo := &phoneComponentRepo{}
	svc := service.New(compRepo, &phoneProjectRepo{}, &stubPhoneAssetRepo{}).SetSourcing(sourcing.NewService(provider))
	server := NewServer(svc, compRepo, &stubPhoneBagRepo{}, 0, activity.NopEmitter, t.TempDir())

	raw := "[)>\x1e06\x1dK37273703\x1d14K017\x1d1P0603SAF1004TCE\x1dQ100\x1d11K087078760\x1d4LTH\x1d1VRoyalohm\x1e\x04"
	scanResp := postScan(t, server, ScanRequest{Vendor: sourcing.ProviderMouser, Format: "data_matrix", RawValue: raw})
	if scanResp.ResolveError != "" {
		t.Fatalf("expected resolved scan, got error %q", scanResp.ResolveError)
	}
	if provider.last.partID != "0603SAF1004TCE" || provider.last.manufacturer != "Royalohm" {
		t.Fatalf("expected manufacturer-aware lookup, got vendor=%s manufacturer=%s part=%s", provider.last.vendor, provider.last.manufacturer, provider.last.partID)
	}

	confirmResp := postConfirm(t, server, ConfirmRequest{ID: scanResp.ID, Quantity: 100})
	if !confirmResp.OK {
		t.Fatalf("expected confirm success, got %#v", confirmResp)
	}
	if compRepo.createdComp == nil {
		t.Fatal("expected component creation on confirm")
	}
	idx := attrsByKeyPhone(compRepo.createdComp.Attributes)
	if attr := idx[registry.AttrResistanceOhms]; attr.Number == nil || *attr.Number != 1e6 {
		t.Fatalf("expected resistance attribute from rich Mouser lookup, got %#v", attr)
	}
	if attr := idx[registry.AttrTempCoPPMC]; attr.Number == nil || *attr.Number != 100 {
		t.Fatalf("expected tempco attribute from rich Mouser lookup, got %#v", attr)
	}
	if attr := idx[registry.AttrPackage]; attr.Text == nil || *attr.Text != "0603" {
		t.Fatalf("expected canonical package from rich Mouser lookup, got %#v", attr)
	}
}

func runPhoneConfirmRegression(t *testing.T, vendor, format, rawValue, wantPartID string, wantQty int, offer sourcing.SupplierOffer) {
	t.Helper()

	provider := &phoneStubProvider{name: vendor, offers: map[string]sourcing.SupplierOffer{vendor + ":" + wantPartID: offer}}
	compRepo := &phoneComponentRepo{}
	svc := service.New(compRepo, &phoneProjectRepo{}, &stubPhoneAssetRepo{}).SetSourcing(sourcing.NewService(provider))
	server := NewServer(svc, compRepo, &stubPhoneBagRepo{}, 0, activity.NopEmitter, t.TempDir())

	scanResp := postScan(t, server, ScanRequest{Vendor: vendor, Format: format, RawValue: rawValue})
	if scanResp.ResolveError != "" {
		t.Fatalf("expected resolved scan, got error %q", scanResp.ResolveError)
	}
	if provider.last.vendor != vendor || provider.last.partID != wantPartID {
		t.Fatalf("expected lookup %s/%s, got %s/%s", vendor, wantPartID, provider.last.vendor, provider.last.partID)
	}
	if scanResp.Quantity != strconv.Itoa(wantQty) {
		t.Fatalf("expected scanned quantity %d, got %q", wantQty, scanResp.Quantity)
	}
	if scanResp.Resolved == nil || scanResp.Resolved.MPN != offer.MPN {
		t.Fatalf("expected resolved component preview, got %#v", scanResp.Resolved)
	}

	confirmResp := postConfirm(t, server, ConfirmRequest{ID: scanResp.ID, Quantity: wantQty})
	if !confirmResp.OK {
		t.Fatalf("expected confirm success, got %#v", confirmResp)
	}
	if compRepo.createdComp == nil {
		t.Fatal("expected component creation on confirm")
	}
	idx := attrsByKeyPhone(compRepo.createdComp.Attributes)
	if attr := idx[registry.AttrResistanceOhms]; attr.Number == nil || *attr.Number != 10000 {
		t.Fatalf("expected resistance attribute, got %#v", attr)
	}
	if attr := idx[registry.AttrPackage]; attr.Text == nil || *attr.Text != "0402" {
		t.Fatalf("expected canonical package attribute, got %#v", attr)
	}
	if compRepo.updatedInventory == nil || compRepo.updatedInventory.Quantity == nil || *compRepo.updatedInventory.Quantity != wantQty {
		t.Fatalf("expected stamped quantity %d, got %#v", wantQty, compRepo.updatedInventory)
	}
	if compRepo.updatedInventory.QuantityMode != domain.QuantityModeExact {
		t.Fatalf("expected exact quantity mode after stamp, got %#v", compRepo.updatedInventory)
	}
	if _, ok := server.pending[scanResp.ID]; ok {
		t.Fatalf("expected pending scan %q to be cleared on confirm", scanResp.ID)
	}
}

func postScan(t *testing.T, server *Server, req ScanRequest) ScanResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal scan request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/scan", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	server.handleScan(resp, httpReq)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected scan 200, got %d", resp.Code)
	}
	var decoded ScanResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode scan response: %v", err)
	}
	return decoded
}

func postConfirm(t *testing.T, server *Server, req ConfirmRequest) ConfirmResponse {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal confirm request: %v", err)
	}
	httpReq := httptest.NewRequest(http.MethodPost, "/confirm", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	server.handleConfirm(resp, httpReq)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected confirm 200, got %d", resp.Code)
	}
	var decoded ConfirmResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode confirm response: %v", err)
	}
	return decoded
}

func attrsByKeyPhone(attrs []domain.AttributeValue) map[string]domain.AttributeValue {
	idx := make(map[string]domain.AttributeValue, len(attrs))
	for _, attr := range attrs {
		idx[attr.Key] = attr
	}
	return idx
}

type phoneComponentRepo struct {
	createdComp      *domain.Component
	getResult        domain.Component
	findResult       []domain.Component
	updatedInventory *domain.Component
	lastFilter       domain.ComponentFilter
	updatedMetadata  *domain.Component
	replacedAttrs    []domain.AttributeValue
	replacedID       string
	components       map[string]domain.Component
	createdOrder     []string
	createCount      int
	upserted         []domain.AttributeDefinition
	deletedIDs       []string
	listCategory     domain.Category
	listResult       []domain.Component
	getErr           error
	findErr          error
	updateErr        error
	replaceErr       error
	deleteErr        error
}

func (r *phoneComponentRepo) CreateComponent(_ context.Context, c domain.Component) (domain.Component, error) {
	copyComp := c
	r.createdComp = &copyComp
	if r.components == nil {
		r.components = make(map[string]domain.Component)
	}
	r.components[c.ID] = c
	return c, nil
}

func (r *phoneComponentRepo) GetComponent(_ context.Context, id string) (domain.Component, error) {
	if r.getErr != nil {
		return domain.Component{}, r.getErr
	}
	if component, ok := r.components[id]; ok {
		return component, nil
	}
	if r.getResult.ID != "" {
		return r.getResult, nil
	}
	return domain.Component{ID: id}, nil
}

func (r *phoneComponentRepo) ListComponentsByCategory(_ context.Context, category domain.Category) ([]domain.Component, error) {
	r.listCategory = category
	return r.listResult, nil
}

func (r *phoneComponentRepo) UpsertAttributeDefinition(_ context.Context, def domain.AttributeDefinition) error {
	r.upserted = append(r.upserted, def)
	return nil
}

func (r *phoneComponentRepo) UpdateComponentMetadata(_ context.Context, c domain.Component) (domain.Component, error) {
	if r.updateErr != nil {
		return domain.Component{}, r.updateErr
	}
	copyComp := c
	r.updatedMetadata = &copyComp
	if r.components == nil {
		r.components = make(map[string]domain.Component)
	}
	r.components[c.ID] = c
	return c, nil
}

func (r *phoneComponentRepo) ReplaceComponentAttributes(_ context.Context, id string, attrs []domain.AttributeValue) error {
	if r.replaceErr != nil {
		return r.replaceErr
	}
	r.replacedID = id
	r.replacedAttrs = attrs
	component := r.components[id]
	component.Attributes = attrs
	r.components[id] = component
	return nil
}

func (r *phoneComponentRepo) FindComponents(_ context.Context, filter domain.ComponentFilter) ([]domain.Component, error) {
	r.lastFilter = filter
	return r.findResult, r.findErr
}

func (r *phoneComponentRepo) UpdateComponentInventory(_ context.Context, c domain.Component) (domain.Component, error) {
	if r.updateErr != nil {
		return domain.Component{}, r.updateErr
	}
	copyComp := c
	r.updatedInventory = &copyComp
	if r.components == nil {
		r.components = make(map[string]domain.Component)
	}
	r.components[c.ID] = c
	return c, nil
}

func (r *phoneComponentRepo) DeleteComponent(_ context.Context, id string) error {
	r.deletedIDs = append(r.deletedIDs, id)
	return r.deleteErr
}

type phoneProjectRepo struct{}

func (phoneProjectRepo) CreateProject(context.Context, domain.Project) (domain.Project, error) {
	return domain.Project{}, nil
}
func (phoneProjectRepo) GetProject(context.Context, string) (domain.Project, error) {
	return domain.Project{}, nil
}
func (phoneProjectRepo) ListProjects(context.Context) ([]domain.Project, error) { return nil, nil }
func (phoneProjectRepo) UpdateProject(context.Context, domain.Project) (domain.Project, error) {
	return domain.Project{}, nil
}
func (phoneProjectRepo) DeleteProject(context.Context, string) error { return nil }
func (phoneProjectRepo) ReplaceProjectRequirements(context.Context, string, []domain.ProjectRequirement) error {
	return nil
}
func (phoneProjectRepo) AddProjectRequirements(context.Context, string, []domain.ProjectRequirement) error {
	return nil
}
func (phoneProjectRepo) SetProjectImportMetadata(context.Context, string, *string, *string, *time.Time) error {
	return nil
}
func (phoneProjectRepo) GetRequirement(context.Context, string) (domain.ProjectRequirement, error) {
	return domain.ProjectRequirement{}, nil
}
func (phoneProjectRepo) SetRequirementResolution(context.Context, string, *domain.RequirementResolution) error {
	return nil
}
func (phoneProjectRepo) AddPartCandidate(context.Context, domain.ProjectPartCandidate) (domain.ProjectPartCandidate, error) {
	return domain.ProjectPartCandidate{}, nil
}
func (phoneProjectRepo) SetPreferredCandidate(context.Context, string, string) error { return nil }
func (phoneProjectRepo) ClearPreferredCandidate(context.Context, string) error       { return nil }
func (phoneProjectRepo) GetPartCandidate(context.Context, string) (domain.ProjectPartCandidate, error) {
	return domain.ProjectPartCandidate{}, nil
}
func (phoneProjectRepo) RemovePartCandidate(context.Context, string) error { return nil }
func (phoneProjectRepo) ListPartCandidates(context.Context, string) ([]domain.ProjectPartCandidate, error) {
	return nil, nil
}
func (phoneProjectRepo) ListPartCandidatesByProject(context.Context, string) ([]domain.ProjectPartCandidate, error) {
	return nil, nil
}
func (phoneProjectRepo) SaveSupplierOffer(context.Context, domain.SavedSupplierOffer) (domain.SavedSupplierOffer, error) {
	return domain.SavedSupplierOffer{}, nil
}
func (phoneProjectRepo) RemoveSavedSupplierOffer(context.Context, string) error { return nil }
func (phoneProjectRepo) ListSavedSupplierOffers(context.Context, string) ([]domain.SavedSupplierOffer, error) {
	return nil, nil
}
func (phoneProjectRepo) ListSavedSupplierOffersByProject(context.Context, string) ([]domain.SavedSupplierOffer, error) {
	return nil, nil
}
func (phoneProjectRepo) LinkSupplierOfferToComponent(context.Context, string, string) error {
	return nil
}
func (phoneProjectRepo) GetSavedSupplierOffer(context.Context, string) (domain.SavedSupplierOffer, error) {
	return domain.SavedSupplierOffer{}, nil
}
func (phoneProjectRepo) UpdatePartCandidateComponent(context.Context, string, string, domain.CandidateOrigin) error {
	return nil
}

type stubPhoneAssetRepo struct{}

func (stubPhoneAssetRepo) CreateComponentAsset(context.Context, domain.ComponentAsset) (domain.ComponentAsset, error) {
	return domain.ComponentAsset{}, nil
}
func (stubPhoneAssetRepo) GetComponentAsset(context.Context, string) (domain.ComponentAsset, error) {
	return domain.ComponentAsset{}, nil
}
func (stubPhoneAssetRepo) ListComponentAssets(context.Context, string) ([]domain.ComponentAsset, error) {
	return nil, nil
}
func (stubPhoneAssetRepo) ListComponentAssetsByType(context.Context, string, domain.AssetType) ([]domain.ComponentAsset, error) {
	return nil, nil
}
func (stubPhoneAssetRepo) UpdateComponentAsset(context.Context, domain.ComponentAsset) (domain.ComponentAsset, error) {
	return domain.ComponentAsset{}, nil
}
func (stubPhoneAssetRepo) DeleteComponentAsset(context.Context, string) error { return nil }
func (stubPhoneAssetRepo) SetSelectedComponentAsset(context.Context, string, domain.AssetType, string) error {
	return nil
}
func (stubPhoneAssetRepo) ClearSelectedComponentAsset(context.Context, string, domain.AssetType) error {
	return nil
}
func (stubPhoneAssetRepo) GetComponentWithAssets(context.Context, string) (domain.ComponentWithAssets, error) {
	return domain.ComponentWithAssets{}, nil
}

type stubPhoneBagRepo struct{}

func (stubPhoneBagRepo) CreateBag(context.Context, domain.InventoryBag) (domain.InventoryBag, error) {
	return domain.InventoryBag{}, nil
}
func (stubPhoneBagRepo) GetBagByQRData(context.Context, string) (domain.InventoryBag, error) {
	return domain.InventoryBag{}, nil
}
func (stubPhoneBagRepo) ListBagsByComponent(context.Context, string) ([]domain.InventoryBag, error) {
	return nil, nil
}
func (stubPhoneBagRepo) DeleteBag(context.Context, string) error              { return nil }
func (stubPhoneBagRepo) FindComponentImageURL(context.Context, string) string { return "" }
func (stubPhoneBagRepo) FindComponentImageURLs(context.Context, []string) map[string]string {
	return map[string]string{}
}
