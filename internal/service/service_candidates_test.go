package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
	"github.com/C-Ma-P/trace/internal/service"
	"github.com/C-Ma-P/trace/internal/sourcing"
)

// --- SelectComponentForRequirement ---

func TestSelectComponentForRequirement_CategoryMatch_Persisted(t *testing.T) {
	proj := &stubProjectRepo{
		getRequirementResult: domain.ProjectRequirement{
			ID:       "req-1",
			Category: domain.CategoryResistor,
		},
	}
	comp := &stubComponentRepo{
		getResult: domain.Component{
			ID:       "cid-1",
			Category: domain.CategoryResistor,
		},
	}
	svc := service.New(comp, proj, &stubAssetRepo{})

	err := svc.SelectComponentForRequirement(context.Background(), "req-1", "cid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.resolvedReqID != "req-1" {
		t.Errorf("expected resolvedReqID %q, got %q", "req-1", proj.resolvedReqID)
	}
	if proj.resolution == nil {
		t.Fatal("expected requirement resolution to be persisted")
	}
	if proj.resolution.Kind != domain.RequirementResolutionKindInternalComponent {
		t.Fatalf("expected internal_component resolution, got %#v", proj.resolution)
	}
	if proj.resolution.ComponentID == nil || *proj.resolution.ComponentID != "cid-1" {
		t.Fatalf("expected resolved component id cid-1, got %#v", proj.resolution)
	}
}

func TestSelectComponentForRequirement_CategoryMismatch_Rejected(t *testing.T) {
	proj := &stubProjectRepo{
		getRequirementResult: domain.ProjectRequirement{
			ID:       "req-1",
			Category: domain.CategoryResistor,
		},
	}
	comp := &stubComponentRepo{
		getResult: domain.Component{
			ID:       "cid-1",
			Category: domain.CategoryCapacitor, // wrong category
		},
	}
	svc := service.New(comp, proj, &stubAssetRepo{})

	err := svc.SelectComponentForRequirement(context.Background(), "req-1", "cid-1")
	if err == nil {
		t.Fatal("expected error for category mismatch")
	}
	var target domain.ErrCategoryMismatch
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrCategoryMismatch, got %T: %v", err, err)
	}
	if proj.resolvedReqID != "" {
		t.Error("repo SetRequirementResolution should not have been called")
	}
}

// --- ClearSelectedComponentForRequirement ---

func TestClearSelectedComponentForRequirement_Delegated(t *testing.T) {
	proj := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	err := svc.ClearSelectedComponentForRequirement(context.Background(), "req-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.resolvedReqID != "req-1" {
		t.Errorf("expected resolvedReqID %q, got %q", "req-1", proj.resolvedReqID)
	}
	if proj.resolution != nil {
		t.Fatalf("expected cleared requirement resolution, got %#v", proj.resolution)
	}
}

func TestAddProviderCandidate_PreservesRawOnSavedOffer(t *testing.T) {
	proj := &stubProjectRepo{
		getRequirementResult: domain.ProjectRequirement{
			ID:        "req-1",
			ProjectID: "proj-1",
			Category:  domain.CategoryResistor,
		},
	}
	raw := map[string]string{
		"Resistance":     "10 kOhms",
		"Package / Case": "0402",
	}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	candidate, err := svc.AddProviderCandidate(context.Background(), "req-1", domain.SavedSupplierOffer{
		Provider:        sourcing.ProviderMouser,
		ProviderPartID:  "603-RC0402FR-0710KL",
		Manufacturer:    "Yageo",
		MPN:             "RC0402FR-0710KL",
		Description:     "General purpose resistor",
		Package:         "Reel",
		Raw:             raw,
		AssetProbeState: "probed",
	}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if candidate.SourceOffer == nil {
		t.Fatal("expected candidate source offer to be hydrated")
	}
	if got := candidate.SourceOffer.Raw["Resistance"]; got != "10 kOhms" {
		t.Fatalf("expected candidate source offer raw resistance, got %#v", candidate.SourceOffer.Raw)
	}
	if candidate.SourceOfferID == nil {
		t.Fatal("expected source offer id")
	}
	saved, ok := proj.savedOffers[*candidate.SourceOfferID]
	if !ok {
		t.Fatalf("expected saved offer %q to be persisted", *candidate.SourceOfferID)
	}
	if got := saved.Raw["Package / Case"]; got != "0402" {
		t.Fatalf("expected raw package spec to be persisted, got %#v", saved.Raw)
	}
}

func TestImportSupplierOffer_PreservesLifecycleAndAttachesDatasheet(t *testing.T) {
	proj := &stubProjectRepo{
		getRequirementResult: domain.ProjectRequirement{
			ID:        "req-1",
			ProjectID: "proj-1",
			Category:  domain.CategoryResistor,
		},
	}
	comp := &stubComponentRepo{}
	assets := &stubAssetRepo{}
	svc := service.New(comp, proj, assets)

	candidate, savedOffer, err := svc.ImportSupplierOffer(context.Background(), "req-1", domain.SavedSupplierOffer{
		Provider:       sourcing.ProviderDigiKey,
		ProviderPartID: "13-RC0603FR-0710KLCT-ND",
		Manufacturer:   "Yageo",
		MPN:            "RC0603FR-0710KL",
		Description:    "General purpose resistor",
		Package:        "Cut Tape",
		Lifecycle:      "Active",
		DatasheetURL:   "https://example.com/yageo-rc0603fr-0710kl.pdf",
		Raw: map[string]string{
			"category":       "Chip Resistors",
			"Resistance":     "10 kOhms",
			"Tolerance":      "1%",
			"Power Rating":   "0.1W",
			"Package / Case": "0603",
		},
	}, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.createdComp == nil || comp.createdComp.Package != "0603" {
		t.Fatalf("expected canonical package on created component, got %#v", comp.createdComp)
	}
	if savedOffer.Lifecycle != "Active" {
		t.Fatalf("expected lifecycle to persist on saved offer, got %#v", savedOffer)
	}
	stored := proj.savedOffers[savedOffer.ID]
	if stored.Lifecycle != "Active" {
		t.Fatalf("expected persisted lifecycle, got %#v", stored)
	}
	if assets.created == nil || assets.created.AssetType != domain.AssetTypeDatasheet || assets.created.Source != "supplier:digikey" {
		t.Fatalf("expected supplier datasheet asset to be created, got %#v", assets.created)
	}
	if assets.created.URLOrPath != "https://example.com/yageo-rc0603fr-0710kl.pdf" {
		t.Fatalf("expected datasheet URL to be attached, got %#v", assets.created)
	}
	if candidate.Component == nil || candidate.Component.SelectedDatasheetAssetID == nil {
		t.Fatalf("expected imported candidate component to carry selected datasheet, got %#v", candidate.Component)
	}
	if proj.resolution == nil || proj.resolution.ComponentID == nil || *proj.resolution.ComponentID != comp.createdComp.ID {
		t.Fatalf("expected preferred saved-offer import to resolve requirement, got %#v", proj.resolution)
	}
	if savedOffer.LinkedComponentID == nil || *savedOffer.LinkedComponentID != comp.createdComp.ID {
		t.Fatalf("expected saved offer linked component id, got %#v", savedOffer.LinkedComponentID)
	}
}

func TestImportProviderCandidate_UsesSavedRawForCanonicalAttrs(t *testing.T) {
	offerID := "offer-1"
	candidateID := "cand-1"
	assets := &stubAssetRepo{}
	proj := &stubProjectRepo{
		getRequirementResult: domain.ProjectRequirement{
			ID:        "req-1",
			ProjectID: "proj-1",
			Category:  domain.CategoryResistor,
		},
		partCandidates: []domain.ProjectPartCandidate{{
			ID:            candidateID,
			ProjectID:     "proj-1",
			RequirementID: "req-1",
			SourceOfferID: &offerID,
			Preferred:     true,
			Origin:        domain.CandidateOriginProvider,
		}},
		savedOffers: map[string]domain.SavedSupplierOffer{
			offerID: {
				ID:             offerID,
				ProjectID:      "proj-1",
				RequirementID:  "req-1",
				Provider:       sourcing.ProviderMouser,
				ProviderPartID: "603-RC0402FR-0710KL",
				Manufacturer:   "Yageo",
				MPN:            "RC0402FR-0710KL",
				Description:    "General purpose resistor",
				Package:        "Reel",
				Lifecycle:      "Active",
				DatasheetURL:   "https://example.com/yageo-rc0402fr-0710kl.pdf",
				Raw: map[string]string{
					"category":                "Chip Resistors",
					"Resistance":              "10 kOhms",
					"Tolerance":               "1%",
					"Power Rating":            "1/10W",
					"Temperature Coefficient": "100ppm/°C",
					"Technology":              "Thick Film",
					"Package / Case":          "0402",
				},
			},
		},
	}
	comp := &stubComponentRepo{}
	svc := service.New(comp, proj, assets)

	imported, err := svc.ImportProviderCandidate(context.Background(), candidateID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.createdComp == nil {
		t.Fatal("expected component creation from provider-backed offer")
	}
	if comp.createdComp.Package != "0402" {
		t.Fatalf("expected canonical package from raw specs, got %q", comp.createdComp.Package)
	}
	idx := comp.createdComp.AttributeIndex()
	if attr, ok := idx[registry.AttrResistanceOhms]; !ok || attr.Number == nil || *attr.Number != 10000 {
		t.Fatalf("expected resistance attribute recovered from saved raw specs, got %#v", comp.createdComp.Attributes)
	}
	if attr, ok := idx[registry.AttrPackage]; !ok || attr.Text == nil || *attr.Text != "0402" {
		t.Fatalf("expected package attribute recovered from saved raw specs, got %#v", comp.createdComp.Attributes)
	}
	if proj.updatedCandidateID != candidateID || proj.updatedCandidateOrigin != domain.CandidateOriginImportedSupplier {
		t.Fatalf("expected provider candidate to be converted to imported supplier, got id=%q origin=%q", proj.updatedCandidateID, proj.updatedCandidateOrigin)
	}
	if proj.linkedOfferID != offerID || proj.linkedComponentID == "" {
		t.Fatalf("expected saved offer to be linked to imported component, got offer=%q component=%q", proj.linkedOfferID, proj.linkedComponentID)
	}
	if imported.Component == nil || imported.Component.Package != "0402" {
		t.Fatalf("expected imported candidate to hydrate canonical component, got %#v", imported.Component)
	}
	if imported.SourceOffer == nil || imported.SourceOffer.Raw["Resistance"] != "10 kOhms" {
		t.Fatalf("expected imported candidate to retain saved raw specs, got %#v", imported.SourceOffer)
	}
	if imported.SourceOffer == nil || imported.SourceOffer.Lifecycle != "Active" {
		t.Fatalf("expected imported candidate to retain lifecycle, got %#v", imported.SourceOffer)
	}
	if assets.created == nil || assets.created.Source != "supplier:mouser" || assets.created.URLOrPath != "https://example.com/yageo-rc0402fr-0710kl.pdf" {
		t.Fatalf("expected provider import to attach supplier datasheet through shared workflow, got %#v", assets.created)
	}
	if proj.resolution == nil || proj.resolution.ComponentID == nil || *proj.resolution.ComponentID != comp.createdComp.ID {
		t.Fatalf("expected preferred provider import to set component resolution, got %#v", proj.resolution)
	}
}
