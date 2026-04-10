package service_test

import (
	"context"
	"errors"
	"testing"

	"componentmanager/internal/domain"
	"componentmanager/internal/service"
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
