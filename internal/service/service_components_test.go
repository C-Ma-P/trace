package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
	"github.com/C-Ma-P/trace/internal/service"
)

func TestSyncCanonicalAttributeDefinitions_AllCategoriesUpserted(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	if err := svc.SyncCanonicalAttributeDefinitions(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(comp.upserted) == 0 {
		t.Fatal("expected definitions to be upserted")
	}

	byCategory := make(map[domain.Category]int)
	for _, def := range comp.upserted {
		byCategory[def.Category]++
	}

	for _, cat := range []domain.Category{domain.CategoryResistor, domain.CategoryCapacitor, domain.CategoryInductor} {
		if byCategory[cat] == 0 {
			t.Errorf("no definitions upserted for category %q", cat)
		}
	}
}

func TestSyncCanonicalAttributeDefinitions_InductorIncluded(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	if err := svc.SyncCanonicalAttributeDefinitions(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var inductorKeys []string
	for _, def := range comp.upserted {
		if def.Category == domain.CategoryInductor {
			inductorKeys = append(inductorKeys, def.Key)
		}
	}

	if len(inductorKeys) == 0 {
		t.Fatal("no inductor definitions were synced")
	}

	wantKeys := map[string]bool{
		"inductance_h": false, "current_a": false, "dcr_ohms": false,
		"tolerance_percent": false, "package": false, "inductor_type": false,
	}
	for _, k := range inductorKeys {
		wantKeys[k] = true
	}
	for key, found := range wantKeys {
		if !found {
			t.Errorf("inductor key %q was not synced", key)
		}
	}
}

// --- UpdateComponentMetadata ---

func TestUpdateComponentMetadata_Persisted(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	c := domain.Component{
		ID:           "cid-1",
		Category:     domain.CategoryResistor,
		MPN:          "RC0402",
		Manufacturer: "Yageo",
		Package:      "0402",
		Description:  "basic resistor",
	}

	result, err := svc.UpdateComponentMetadata(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.updatedComp == nil {
		t.Fatal("expected UpdateComponentMetadata to be called on repository")
	}
	if result.ID != c.ID {
		t.Errorf("expected ID %q, got %q", c.ID, result.ID)
	}
	if comp.updatedComp.Category != domain.CategoryResistor {
		t.Error("expected category to be unchanged")
	}
}

// --- ReplaceComponentAttributes ---

func TestReplaceComponentAttributes_InvalidAttr_Rejected(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{
			ID:       "cid-1",
			Category: domain.CategoryResistor,
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	n := 47.0
	attrs := []domain.AttributeValue{
		// wrong unit for resistance (should be "ohm" but passing "milli-ohm")
		{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Number: &n, Unit: "milli-ohm"},
	}

	err := svc.ReplaceComponentAttributes(context.Background(), "cid-1", attrs)
	if err == nil {
		t.Fatal("expected error for unit mismatch")
	}
	var target domain.ErrAttributeUnitMismatch
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrAttributeUnitMismatch, got %T: %v", err, err)
	}
	if comp.replacedID != "" {
		t.Error("repository ReplaceComponentAttributes should not have been called")
	}
}

func TestReplaceComponentAttributes_Valid(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{
			ID:       "cid-1",
			Category: domain.CategoryResistor,
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	n := 10000.0
	attrs := []domain.AttributeValue{
		{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Number: &n, Unit: "ohm"},
	}

	err := svc.ReplaceComponentAttributes(context.Background(), "cid-1", attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.replacedID != "cid-1" {
		t.Errorf("expected replacedID %q, got %q", "cid-1", comp.replacedID)
	}
	if len(comp.replacedAttrs) != 1 {
		t.Errorf("expected 1 attr passed to repo, got %d", len(comp.replacedAttrs))
	}
}

func TestReplaceComponentAttributes_UnknownKey_Rejected(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{
			ID:       "cid-1",
			Category: domain.CategoryCapacitor,
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	n := 1.0
	attrs := []domain.AttributeValue{
		{Key: "not_a_real_key", ValueType: domain.ValueTypeNumber, Number: &n},
	}

	err := svc.ReplaceComponentAttributes(context.Background(), "cid-1", attrs)
	if err == nil {
		t.Fatal("expected error for unknown attribute key")
	}
	var target domain.ErrUnknownAttribute
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownAttribute, got %T: %v", err, err)
	}
}

// --- FindComponents ---

func TestFindComponents_EmptyFilter_DelegatesToRepo(t *testing.T) {
	comp := &stubComponentRepo{
		findResult: []domain.Component{
			{ID: "c1", Category: domain.CategoryResistor, MPN: "R1"},
			{ID: "c2", Category: domain.CategoryCapacitor, MPN: "C1"},
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	results, err := svc.FindComponents(context.Background(), domain.ComponentFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestFindComponents_ByCategory(t *testing.T) {
	cat := domain.CategoryResistor
	comp := &stubComponentRepo{
		findResult: []domain.Component{
			{ID: "c1", Category: domain.CategoryResistor, MPN: "R1"},
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{Category: &cat}
	_, err := svc.FindComponents(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.lastFilter.Category == nil || *comp.lastFilter.Category != domain.CategoryResistor {
		t.Error("expected category filter to be passed to repository")
	}
}

func TestFindComponents_ByManufacturer(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{Manufacturer: "Yageo"}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.Manufacturer != "Yageo" {
		t.Errorf("expected Manufacturer %q passed to repo, got %q", "Yageo", comp.lastFilter.Manufacturer)
	}
}

func TestFindComponents_ByMPN(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{MPN: "RC04"}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.MPN != "RC04" {
		t.Errorf("expected MPN %q passed to repo, got %q", "RC04", comp.lastFilter.MPN)
	}
}

func TestFindComponents_ByPackage(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{Package: "0402"}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.Package != "0402" {
		t.Errorf("expected Package %q passed to repo, got %q", "0402", comp.lastFilter.Package)
	}
}

func TestFindComponents_ByText(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{Text: "ceramic"}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.Text != "ceramic" {
		t.Errorf("expected Text %q passed to repo, got %q", "ceramic", comp.lastFilter.Text)
	}
}

func TestFindComponents_CombinedFilter(t *testing.T) {
	cat := domain.CategoryCapacitor
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{
		Category:     &cat,
		Manufacturer: "Murata",
		Package:      "0402",
	}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.Manufacturer != "Murata" || comp.lastFilter.Package != "0402" {
		t.Error("combined filter fields not passed to repository")
	}
}

func TestFindComponents_ReturnsAttributesFromRepo(t *testing.T) {
	n := 100.0
	comp := &stubComponentRepo{
		findResult: []domain.Component{
			{
				ID:       "c1",
				Category: domain.CategoryResistor,
				Attributes: []domain.AttributeValue{
					{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Number: &n, Unit: "ohm"},
				},
			},
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	results, err := svc.FindComponents(context.Background(), domain.ComponentFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if len(results[0].Attributes) == 0 {
		t.Error("expected attributes to be present on returned components")
	}
}

// --- ErrNotFound propagation ---

func TestGetComponent_ErrNotFound(t *testing.T) {
	comp := &stubComponentRepo{getErr: domain.ErrNotFound{ID: "cid-x"}}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.GetComponent(context.Background(), "cid-x")
	var target domain.ErrNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrNotFound, got %T: %v", err, err)
	}
	if target.ID != "cid-x" {
		t.Errorf("expected ID %q, got %q", "cid-x", target.ID)
	}
}

func TestUpdateComponentMetadata_ErrNotFound(t *testing.T) {
	comp := &stubComponentRepo{updateCompErr: domain.ErrNotFound{ID: "cid-x"}}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.UpdateComponentMetadata(context.Background(), domain.Component{ID: "cid-x", MPN: "X"})
	var target domain.ErrNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrNotFound, got %T: %v", err, err)
	}
	if target.ID != "cid-x" {
		t.Errorf("expected ID %q, got %q", "cid-x", target.ID)
	}
}
