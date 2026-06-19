package service_test

import (
	"context"
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
	"github.com/C-Ma-P/trace/internal/service"
	"github.com/C-Ma-P/trace/internal/sourcing"
)

type capturingProvider struct {
	query  sourcing.RequirementQuery
	offers []sourcing.SupplierOffer
}

func (p *capturingProvider) Name() string {
	return "capture"
}

func (p *capturingProvider) Enabled() bool {
	return true
}

func (p *capturingProvider) Search(_ context.Context, query sourcing.RequirementQuery) ([]sourcing.SupplierOffer, error) {
	p.query = query
	return p.offers, nil
}

func (p *capturingProvider) FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestSourceRequirement_UsesSelectedComponentSignals(t *testing.T) {
	componentID := "comp-1"
	provider := &capturingProvider{offers: []sourcing.SupplierOffer{{Provider: "capture", MPN: "RC0402FR-0710KL"}}}
	compRepo := &stubComponentRepo{getResult: domain.Component{
		ID:           componentID,
		Category:     domain.CategoryResistor,
		Manufacturer: "Yageo",
		MPN:          "RC0402FR-0710KL",
		Package:      "0402",
		Description:  "Chip resistor",
	}}
	projRepo := &stubProjectRepo{getRequirementResult: domain.ProjectRequirement{
		ID:                  "req-1",
		Name:                "Pull-up resistor",
		Category:            domain.CategoryResistor,
		Quantity:            4,
		SelectedComponentID: &componentID,
	}}
	svc := service.New(compRepo, projRepo, &stubAssetRepo{}).SetSourcing(sourcing.NewService(provider))

	result, err := svc.SourceRequirement(context.Background(), "req-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.query.MPN != "RC0402FR-0710KL" {
		t.Fatalf("expected selected component MPN in query, got %#v", provider.query)
	}
	if provider.query.Manufacturer != "Yageo" {
		t.Fatalf("expected selected component manufacturer in query, got %#v", provider.query)
	}
	if len(result.Offers) != 1 {
		t.Fatalf("expected one offer, got %#v", result.Offers)
	}
}

func TestSourceRequirement_UsesResolutionModel(t *testing.T) {
	componentID := "comp-2"
	provider := &capturingProvider{offers: []sourcing.SupplierOffer{{Provider: "capture", MPN: "GRM155R71C104KA88D"}}}
	compRepo := &stubComponentRepo{getResult: domain.Component{
		ID:           componentID,
		Category:     domain.CategoryCapacitor,
		Manufacturer: "Murata",
		MPN:          "GRM155R71C104KA88D",
		Package:      "0402",
	}}
	projRepo := &stubProjectRepo{getRequirementResult: domain.ProjectRequirement{
		ID:       "req-2",
		Name:     "Bypass cap",
		Category: domain.CategoryCapacitor,
		Quantity: 10,
		Resolution: &domain.RequirementResolution{
			Kind:        domain.RequirementResolutionKindInternalComponent,
			ComponentID: &componentID,
		},
	}}
	svc := service.New(compRepo, projRepo, &stubAssetRepo{}).SetSourcing(sourcing.NewService(provider))

	result, err := svc.SourceRequirement(context.Background(), "req-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.query.MPN != "GRM155R71C104KA88D" {
		t.Fatalf("expected resolved component MPN in query, got %q", provider.query.MPN)
	}
	if provider.query.Manufacturer != "Murata" {
		t.Fatalf("expected resolved component manufacturer in query, got %q", provider.query.Manufacturer)
	}
	if len(result.Offers) != 1 {
		t.Fatalf("expected one offer, got %d", len(result.Offers))
	}
}

func TestSourceRequirement_NoResolution_NoSelectedComponent(t *testing.T) {
	provider := &capturingProvider{offers: []sourcing.SupplierOffer{}}
	compRepo := &stubComponentRepo{}
	projRepo := &stubProjectRepo{getRequirementResult: domain.ProjectRequirement{
		ID:       "req-3",
		Name:     "Generic resistor",
		Category: domain.CategoryResistor,
		Quantity: 1,
	}}
	svc := service.New(compRepo, projRepo, &stubAssetRepo{}).SetSourcing(sourcing.NewService(provider))

	_, err := svc.SourceRequirement(context.Background(), "req-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.query.SelectedComponent != nil {
		t.Error("expected no selected component in query")
	}
}

func TestResolveComponentFromOffer_CreatesComponentWithSupplierAttrs(t *testing.T) {
	compRepo := &stubComponentRepo{}
	assets := &stubAssetRepo{}
	svc := service.New(compRepo, &stubProjectRepo{}, assets)

	component, err := svc.ResolveComponentFromOffer(context.Background(), sourcing.SupplierOffer{
		Provider:     sourcing.ProviderMouser,
		Manufacturer: "Yageo",
		MPN:          "RC0402FR-0710KL",
		Package:      "Reel",
		Description:  "10k 1% thick film resistor",
		DatasheetURL: "https://example.com/rc0402.pdf",
		Raw: map[string]string{
			"category":                "Chip Resistors",
			"Resistance":              "10 kOhms",
			"Tolerance":               "1%",
			"Power Rating":            "0.1W",
			"Temperature Coefficient": "100ppm/°C",
			"Technology":              "Thick Film",
			"Package / Case":          "0402",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compRepo.createdComp == nil {
		t.Fatal("expected component creation")
	}
	if component.Category != domain.CategoryResistor {
		t.Fatalf("expected resistor category, got %s", component.Category)
	}
	if component.Package != "0402" {
		t.Fatalf("expected canonical component package, got %q", component.Package)
	}
	idx := component.AttributeIndex()
	if attr, ok := idx[registry.AttrResistanceOhms]; !ok || attr.Number == nil || *attr.Number != 10000 {
		t.Fatalf("expected resistance attribute on created component, got %#v", component.Attributes)
	}
	if attr, ok := idx[registry.AttrPackage]; !ok || attr.Text == nil || *attr.Text != "0402" {
		t.Fatalf("expected package attribute on created component, got %#v", component.Attributes)
	}
	if compRepo.replacedID != "" {
		t.Fatalf("did not expect ReplaceComponentAttributes on create, got %q", compRepo.replacedID)
	}
	if assets.created == nil || assets.created.URLOrPath != "https://example.com/rc0402.pdf" {
		t.Fatalf("expected datasheet asset creation, got %#v", assets.created)
	}
	if assets.setAssetType != domain.AssetTypeDatasheet {
		t.Fatalf("expected datasheet asset to be selected, got %q", assets.setAssetType)
	}
}

func TestResolveComponentFromOffer_ReusesComponentAndMergesMissingAttrs(t *testing.T) {
	existingTolerance := 5.0
	findID := "comp-1"
	compRepo := &stubComponentRepo{
		findResult: []domain.Component{{ID: findID, Category: domain.CategoryResistor, Manufacturer: "Yageo", MPN: "RC0402FR-0710KL"}},
		getResult: domain.Component{
			ID:           findID,
			Category:     domain.CategoryResistor,
			Manufacturer: "Yageo",
			MPN:          "RC0402FR-0710KL",
			Package:      "Reel",
			Attributes: []domain.AttributeValue{{
				Key:       registry.AttrTolerancePercent,
				ValueType: domain.ValueTypeNumber,
				Number:    &existingTolerance,
				Unit:      "percent",
			}},
		},
	}
	assets := &stubAssetRepo{}
	svc := service.New(compRepo, &stubProjectRepo{}, assets)

	component, err := svc.ResolveComponentFromOffer(context.Background(), sourcing.SupplierOffer{
		Provider:     sourcing.ProviderMouser,
		Manufacturer: "Yageo",
		MPN:          "RC0402FR-0710KL",
		Package:      "0402",
		Description:  "10k 1% thick film resistor",
		DatasheetURL: "https://example.com/rc0402.pdf",
		Raw: map[string]string{
			"category":                "Chip Resistors",
			"Resistance":              "10 kOhms",
			"Tolerance":               "1%",
			"Power Rating":            "0.1W",
			"Temperature Coefficient": "100ppm/°C",
			"Technology":              "Thick Film",
			"Package / Case":          "0402",
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if compRepo.createdComp != nil {
		t.Fatal("did not expect component creation when match exists")
	}
	if compRepo.replacedID != findID {
		t.Fatalf("expected ReplaceComponentAttributes for %q, got %q", findID, compRepo.replacedID)
	}
	if compRepo.updatedComp == nil || compRepo.updatedComp.Package != "0402" {
		t.Fatalf("expected metadata package to be repaired from reel to 0402, got %#v", compRepo.updatedComp)
	}
	merged := attrsByKey(compRepo.replacedAttrs)
	if attr := merged[registry.AttrTolerancePercent]; attr.Number == nil || *attr.Number != 5 {
		t.Fatalf("expected existing tolerance to be preserved, got %#v", attr)
	}
	if attr := merged[registry.AttrResistanceOhms]; attr.Number == nil || *attr.Number != 10000 {
		t.Fatalf("expected missing resistance to be added, got %#v", attr)
	}
	if len(component.Attributes) != len(compRepo.replacedAttrs) {
		t.Fatalf("expected returned component attrs to reflect merged set, got %#v vs %#v", component.Attributes, compRepo.replacedAttrs)
	}
	if assets.created == nil || assets.created.URLOrPath != "https://example.com/rc0402.pdf" {
		t.Fatalf("expected datasheet asset creation for reused component, got %#v", assets.created)
	}
}

func attrsByKey(attrs []domain.AttributeValue) map[string]domain.AttributeValue {
	idx := make(map[string]domain.AttributeValue, len(attrs))
	for _, attr := range attrs {
		idx[attr.Key] = attr
	}
	return idx
}
