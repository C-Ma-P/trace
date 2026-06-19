package sourcing

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	easyeda "github.com/C-Ma-P/go-easyeda"
	digikey "github.com/PatrickWalther/go-digikey"
	lcsc "github.com/PatrickWalther/go-lcsc"
	mouser "github.com/PatrickWalther/go-mouser"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
)

type stubProvider struct {
	name    string
	enabled bool
	offers  []SupplierOffer
	err     error
}

func (s stubProvider) Name() string {
	return s.name
}

func (s stubProvider) Enabled() bool {
	return s.enabled
}

func (s stubProvider) Search(_ context.Context, _ RequirementQuery) ([]SupplierOffer, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.offers, nil
}

func (s stubProvider) FriendlyError(err error) string {
	return err.Error()
}

func TestBuildRequirementQuery_PrefersSelectedComponentSignals(t *testing.T) {
	resistance := 10000.0
	manufacturer := "Vishay"
	packageName := "0402"
	req := domain.ProjectRequirement{
		ID:       "req-1",
		Name:     "Pull-up resistor",
		Category: domain.CategoryResistor,
		Quantity: 4,
		Constraints: []domain.RequirementConstraint{
			{Key: registry.AttrManufacturer, ValueType: domain.ValueTypeText, Operator: domain.OperatorEqual, Text: &manufacturer},
			{Key: registry.AttrPackage, ValueType: domain.ValueTypeText, Operator: domain.OperatorEqual, Text: &packageName},
			{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Operator: domain.OperatorEqual, Number: &resistance, Unit: "ohm"},
		},
	}
	selected := &domain.Component{
		ID:           "comp-1",
		Category:     domain.CategoryResistor,
		Manufacturer: "Yageo",
		MPN:          "RC0402FR-0710KL",
		Package:      "0402",
		Description:  "Thick film chip resistor",
	}

	query := BuildRequirementQuery(req, selected)

	if query.Manufacturer != "Yageo" {
		t.Fatalf("expected selected manufacturer to win, got %q", query.Manufacturer)
	}
	if query.MPN != "RC0402FR-0710KL" {
		t.Fatalf("expected selected MPN, got %q", query.MPN)
	}
	if query.Package != "0402" {
		t.Fatalf("expected selected package, got %q", query.Package)
	}
	if !containsString(query.ValueTerms, "10k") {
		t.Fatalf("expected resistor value hint, got %#v", query.ValueTerms)
	}
	if len(query.SearchTerms) == 0 || !strings.Contains(strings.ToLower(query.SearchTerms[0]), "yageo") {
		t.Fatalf("expected exact selected-component search term first, got %#v", query.SearchTerms)
	}
}

func TestNormalizeProviderOffers(t *testing.T) {
	digi := normalizeDigiKeyProduct(digikey.Product{
		Manufacturer:              digikey.Manufacturer{Name: "Texas Instruments"},
		ManufacturerProductNumber: "SN74LVC1G14DBVR",
		DigiKeyProductNumber:      "296-8480-1-ND",
		DetailedDescription:       "IC INVERTER SCHMITT SOT23-5",
		ProductURL:                "https://www.digikey.com/en/products/detail/test",
		DatasheetURL:              "https://example.com/datasheet.pdf",
		PhotoURL:                  "https://example.com/photo.jpg",
		QuantityAvailable:         1234,
		UnitPrice:                 0.12,
		StandardPackage:           1,
		ProductStatus:             digikey.ProductStatus{Text: "Active"},
		Parameters: []digikey.Parameter{
			{ParameterText: "Package / Case", ValueText: "SOT-23-5"},
			{ParameterText: "Logic Type", ValueText: "Inverter"},
		},
		ProductVariations: []digikey.ProductVariation{{
			PackageType:          digikey.PackageType{Name: "Tape & Reel (TR)"},
			MinimumOrderQuantity: 1,
		}},
	})
	if digi.Provider != ProviderDigiKey || digi.SupplierPartNumber != "296-8480-1-ND" {
		t.Fatalf("unexpected DigiKey normalization: %#v", digi)
	}
	if digi.ImageURL != "https://example.com/photo.jpg" {
		t.Fatalf("expected Digikey ImageURL to be populated, got %q", digi.ImageURL)
	}
	if got := digi.Raw["Logic Type"]; got != "Inverter" {
		t.Fatalf("expected DigiKey parameter in raw map, got %q", got)
	}

	mouserOffer := normalizeMouserPart(mouser.Part{
		Manufacturer:           "Murata",
		ManufacturerPartNumber: "GRM155R71C104KA88D",
		MouserPartNumber:       "81-GRM155R71C104KA8D",
		Description:            "Multilayer Ceramic Capacitors MLCC - SMD/SMT 0.1uF 16V X7R 10%",
		DataSheetUrl:           "https://example.com/mlcc.pdf",
		ProductDetailUrl:       "https://www.mouser.com/ProductDetail/test",
		AvailabilityInStock:    "5,432 In Stock",
		Min:                    "1",
		Category:               "Chip Resistors",
		LifecycleStatus:        "Active",
		PriceBreaks:            []mouser.PriceBreak{{Quantity: 1, Price: "$0.02", Currency: "USD"}},
		ProductAttributes: []mouser.ProductAttribute{
			{AttributeName: "Packaging", AttributeValue: "Reel"},
			{AttributeName: "Package / Case", AttributeValue: "0402"},
			{AttributeName: "Tolerance", AttributeValue: "1%"},
		},
	})
	if mouserOffer.Stock == nil || *mouserOffer.Stock != 5432 || mouserOffer.Package != "0402" {
		t.Fatalf("unexpected Mouser normalization: %#v", mouserOffer)
	}
	if got := mouserOffer.Raw["Tolerance"]; got != "1%" {
		t.Fatalf("expected Mouser attribute in raw map, got %q", got)
	}
	if got := mouserOffer.Raw["category"]; got != "Chip Resistors" {
		t.Fatalf("expected Mouser category to remain in raw map, got %q", got)
	}

	lcscOffer := normalizeLCSCProduct(lcsc.Product{
		BrandNameEn:       "TDK",
		ProductModel:      "C1005X7R1C104K050BB",
		ProductCode:       "C14663",
		ProductIntroEn:    "100nF 16V 0402 X7R",
		PdfURL:            "https://example.com/tdk.pdf",
		StockNumber:       9999,
		MinPacketNumber:   5,
		EncapStandard:     "0402",
		CatalogName:       "Chip Resistor - Surface Mount",
		ParentCatalogName: "Resistors",
		ParamVOList: []lcsc.Parameter{
			{ParamNameEn: "Resistance", ParamValueEn: "10 kOhms"},
			{ParamNameEn: "Tolerance", ParamValueEn: "1%"},
		},
		ProductPriceList: []lcsc.PriceBreak{{Ladder: 1, ProductPrice: lcsc.FlexFloat64(0.01), CurrencySymbol: "USD"}},
	})
	if lcscOffer.Provider != ProviderLCSC || lcscOffer.MOQ == nil || *lcscOffer.MOQ != 5 {
		t.Fatalf("unexpected LCSC normalization: %#v", lcscOffer)
	}
	if got := lcscOffer.Raw["Resistance"]; got != "10 kOhms" {
		t.Fatalf("expected LCSC detail parameter in raw map, got %q", got)
	}
	if got := lcscOffer.Raw["catalog"]; got != "Chip Resistor - Surface Mount" {
		t.Fatalf("expected LCSC catalog to remain in raw map, got %q", got)
	}
}

func TestComponentAttributesFromOffer_Resistor(t *testing.T) {
	offer := SupplierOffer{
		Provider:    ProviderMouser,
		Description: "10k 1% 1/10W thick film resistor 0402 100ppm/°C",
		Package:     "0402",
		Raw: map[string]string{
			"category":                "Chip Resistors",
			"Resistance":              "10 kOhms",
			"Tolerance":               "±1%",
			"Power Rating":            "1/10W",
			"Temperature Coefficient": "100ppm/°C",
			"Technology":              "Thick Film",
			"Package / Case":          "0402",
		},
	}

	attrs := ComponentAttributesFromOffer(domain.CategoryResistor, offer)
	idx := attrsByKey(attrs)

	assertNumberAttr(t, idx, registry.AttrResistanceOhms, 10000, "ohm")
	assertNumberAttr(t, idx, registry.AttrTolerancePercent, 1, "percent")
	assertNumberAttr(t, idx, registry.AttrPowerW, 0.1, "W")
	assertNumberAttr(t, idx, registry.AttrTempCoPPMC, 100, "ppm/C")
	assertTextAttr(t, idx, registry.AttrResistorType, "Thick Film")
	assertTextAttr(t, idx, registry.AttrPackage, "0402")
	if len(attrs) != 6 {
		t.Fatalf("expected 6 canonical attrs, got %#v", attrs)
	}
}

func TestComponentAttributesFromOffer_Resistor_FuzzyMouserFieldsAndDescription(t *testing.T) {
	offer := SupplierOffer{
		Provider:    ProviderMouser,
		Description: "Res Thick Film 0603 1M Ohms 1% 1/10W 100 ppm/C",
		Package:     "Reel",
		Raw: map[string]string{
			"category":             "Chip Resistors",
			"Ohms":                 "1 MOhms",
			"TCR (ppm/C)":          "100 ppm/C",
			"Packaging":            "Reel",
			"Case Code - in":       "0603",
			"Technology":           "Thick Film",
			"Power (Watts)":        "1/10W",
			"Resistance Tolerance": "1%",
		},
	}

	attrs := ComponentAttributesFromOffer(domain.CategoryResistor, offer)
	idx := attrsByKey(attrs)

	assertNumberAttr(t, idx, registry.AttrResistanceOhms, 1e6, "ohm")
	assertNumberAttr(t, idx, registry.AttrTempCoPPMC, 100, "ppm/C")
	assertTextAttr(t, idx, registry.AttrPackage, "0603")
}

func attrsByKey(attrs []domain.AttributeValue) map[string]domain.AttributeValue {
	idx := make(map[string]domain.AttributeValue, len(attrs))
	for _, attr := range attrs {
		idx[attr.Key] = attr
	}
	return idx
}

func assertNumberAttr(t *testing.T, idx map[string]domain.AttributeValue, key string, want float64, unit string) {
	t.Helper()
	attr, ok := idx[key]
	if !ok || attr.Number == nil {
		t.Fatalf("expected numeric attr %q, got %#v", key, attr)
	}
	if *attr.Number != want || attr.Unit != unit {
		t.Fatalf("expected %s=%v %s, got %#v", key, want, unit, attr)
	}
}

func assertTextAttr(t *testing.T, idx map[string]domain.AttributeValue, key, want string) {
	t.Helper()
	attr, ok := idx[key]
	if !ok || attr.Text == nil {
		t.Fatalf("expected text attr %q, got %#v", key, attr)
	}
	if *attr.Text != want {
		t.Fatalf("expected %s=%q, got %#v", key, want, attr)
	}
}

func TestRankOffers_PrefersExactMatches(t *testing.T) {
	query := RequirementQuery{
		Category:     domain.CategoryCapacitor,
		Manufacturer: "Murata",
		MPN:          "GRM155R71C104KA88D",
		Package:      "0402",
		ValueTerms:   []string{"100nF"},
		TextTerms:    []string{"X7R"},
	}
	offers := RankOffers(query, []SupplierOffer{
		{
			Provider:     ProviderMouser,
			Manufacturer: "Murata",
			MPN:          "GRM155R71C104KA88D",
			Description:  "100nF 16V X7R capacitor",
			Package:      "0402",
		},
		{
			Provider:     ProviderLCSC,
			Manufacturer: "Another",
			MPN:          "XYZ123",
			Description:  "100nF capacitor",
			Package:      "0603",
		},
	})

	if offers[0].Provider != ProviderMouser {
		t.Fatalf("expected exact offer first, got %#v", offers)
	}
	if offers[0].MatchScore <= offers[1].MatchScore {
		t.Fatalf("expected exact offer to outscore fuzzy offer: %#v", offers)
	}
	if !containsString(offers[0].MatchReasons, "Exact MPN match") {
		t.Fatalf("expected exact match reason, got %#v", offers[0].MatchReasons)
	}
}

func TestService_Source_ProviderFailureSoftens(t *testing.T) {
	svc := NewService(
		stubProvider{name: ProviderDigiKey, enabled: true, err: errors.New("timeout")},
		stubProvider{name: ProviderMouser, enabled: true, offers: []SupplierOffer{{Provider: ProviderMouser, MPN: "ABC123"}}},
		stubProvider{name: ProviderLCSC, enabled: false},
	)

	result := svc.Source(context.Background(), RequirementQuery{Category: domain.CategoryIntegratedCircuit})

	if len(result.Offers) != 1 {
		t.Fatalf("expected one surviving offer, got %#v", result.Offers)
	}
	if len(result.Providers) != 3 {
		t.Fatalf("expected provider statuses for all providers, got %#v", result.Providers)
	}
	if result.Providers[0].Status != "error" {
		t.Fatalf("expected first provider to fail softly, got %#v", result.Providers[0])
	}
	if result.Providers[1].Status != "success" {
		t.Fatalf("expected second provider success, got %#v", result.Providers[1])
	}
	if result.Providers[2].Status != "disabled" {
		t.Fatalf("expected disabled provider status, got %#v", result.Providers[2])
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestLCSCProvider_EnrichOfferAssets(t *testing.T) {
	provider := &LCSCProvider{
		bundleFetcher: func(_ context.Context, lcscID string, _ easyeda.FetchOptions) (*easyeda.ComponentBundle, error) {
			if lcscID != "C12345" {
				t.Fatalf("unexpected LCSC ID: %s", lcscID)
			}
			return &easyeda.ComponentBundle{
				Extracted: &easyeda.ComponentMetadata{
					SymbolRaw:    json.RawMessage(`{"pins":[]}`),
					FootprintRaw: json.RawMessage(`{"pads":[]}`),
					DatasheetURL: "https://example.com/datasheet.pdf",
				},
			}, nil
		},
	}

	offer := SupplierOffer{SupplierPartNumber: "C12345"}
	provider.enrichOfferAssets(context.Background(), &offer)

	if !offer.HasSymbol {
		t.Fatalf("expected HasSymbol=true")
	}
	if !offer.HasFootprint {
		t.Fatalf("expected HasFootprint=true")
	}
	if !offer.HasDatasheet {
		t.Fatalf("expected HasDatasheet=true")
	}
	if offer.DatasheetURL != "https://example.com/datasheet.pdf" {
		t.Fatalf("expected datasheet url to be preserved, got %q", offer.DatasheetURL)
	}
}
