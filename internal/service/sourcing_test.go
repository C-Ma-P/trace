package service_test

import (
	"context"
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
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
