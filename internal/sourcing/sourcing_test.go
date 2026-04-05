package sourcing

import (
	"context"
	"errors"
	"strings"
	"testing"

	digikey "github.com/PatrickWalther/go-digikey"
	lcsc "github.com/PatrickWalther/go-lcsc"
	mouser "github.com/PatrickWalther/go-mouser"

	"componentmanager/internal/domain"
	"componentmanager/internal/domain/registry"
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
		QuantityAvailable:         1234,
		UnitPrice:                 0.12,
		StandardPackage:           1,
		ProductStatus:             digikey.ProductStatus{Text: "Active"},
		ProductVariations: []digikey.ProductVariation{{
			PackageType:          digikey.PackageType{Name: "Tape & Reel (TR)"},
			MinimumOrderQuantity: 1,
		}},
	}, "USD")
	if digi.Provider != "DigiKey" || digi.SupplierPartNumber != "296-8480-1-ND" || digi.Currency != "USD" {
		t.Fatalf("unexpected DigiKey normalization: %#v", digi)
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
		LifecycleStatus:        "Active",
		PriceBreaks:            []mouser.PriceBreak{{Quantity: 1, Price: "$0.02", Currency: "USD"}},
		ProductAttributes:      []mouser.ProductAttribute{{AttributeName: "Package / Case", AttributeValue: "0402"}},
	})
	if mouserOffer.Stock == nil || *mouserOffer.Stock != 5432 || mouserOffer.Package != "0402" {
		t.Fatalf("unexpected Mouser normalization: %#v", mouserOffer)
	}

	lcscOffer := normalizeLCSCProduct(lcsc.Product{
		BrandNameEn:      "TDK",
		ProductModel:     "C1005X7R1C104K050BB",
		ProductCode:      "C14663",
		ProductIntroEn:   "100nF 16V 0402 X7R",
		PdfURL:           "https://example.com/tdk.pdf",
		StockNumber:      9999,
		MinPacketNumber:  5,
		EncapStandard:    "0402",
		ProductPriceList: []lcsc.PriceBreak{{Ladder: 1, ProductPrice: lcsc.FlexFloat64(0.01), CurrencySymbol: "USD"}},
	})
	if lcscOffer.Provider != "LCSC" || lcscOffer.MOQ == nil || *lcscOffer.MOQ != 5 {
		t.Fatalf("unexpected LCSC normalization: %#v", lcscOffer)
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
			Provider:     "Mouser",
			Manufacturer: "Murata",
			MPN:          "GRM155R71C104KA88D",
			Description:  "100nF 16V X7R capacitor",
			Package:      "0402",
		},
		{
			Provider:     "LCSC",
			Manufacturer: "Another",
			MPN:          "XYZ123",
			Description:  "100nF capacitor",
			Package:      "0603",
		},
	})

	if offers[0].Provider != "Mouser" {
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
		stubProvider{name: "DigiKey", enabled: true, err: errors.New("timeout")},
		stubProvider{name: "Mouser", enabled: true, offers: []SupplierOffer{{Provider: "Mouser", MPN: "ABC123"}}},
		stubProvider{name: "LCSC", enabled: false},
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
