package domain_test

import (
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
)

func strPtr(s string) *string { return &s }

// --- RequirementResolution ---

func TestNewComponentRequirementResolution_Valid(t *testing.T) {
	r := domain.NewComponentRequirementResolution("cid-1")
	if r == nil {
		t.Fatal("expected non-nil resolution")
	}
	if r.Kind != domain.RequirementResolutionKindInternalComponent {
		t.Errorf("expected kind %q, got %q", domain.RequirementResolutionKindInternalComponent, r.Kind)
	}
	if r.ComponentID == nil || *r.ComponentID != "cid-1" {
		t.Errorf("expected componentID cid-1, got %v", r.ComponentID)
	}
}

func TestNewComponentRequirementResolution_Empty(t *testing.T) {
	r := domain.NewComponentRequirementResolution("")
	if r != nil {
		t.Fatalf("expected nil for empty component id, got %#v", r)
	}
}

func TestNewComponentRequirementResolution_Whitespace(t *testing.T) {
	r := domain.NewComponentRequirementResolution("   ")
	if r != nil {
		t.Fatalf("expected nil for whitespace-only component id, got %#v", r)
	}
}

func TestRequirementResolution_Normalize_TrimsWhitespace(t *testing.T) {
	cid := "  cid-1  "
	r := &domain.RequirementResolution{
		Kind:        "  internal_component  ",
		ComponentID: &cid,
	}
	r.Normalize()
	if r.Kind != domain.RequirementResolutionKindInternalComponent {
		t.Errorf("expected trimmed kind, got %q", r.Kind)
	}
	if r.ComponentID == nil || *r.ComponentID != "cid-1" {
		t.Errorf("expected trimmed componentID, got %v", r.ComponentID)
	}
}

func TestRequirementResolution_Normalize_ClearsEmptyComponentID(t *testing.T) {
	cid := "   "
	r := &domain.RequirementResolution{
		Kind:        domain.RequirementResolutionKindInternalComponent,
		ComponentID: &cid,
	}
	r.Normalize()
	// internal_component with no componentID normalizes kind to empty
	if r.Kind != "" {
		t.Errorf("expected empty kind when component ID is blank, got %q", r.Kind)
	}
	if r.ComponentID != nil {
		t.Errorf("expected nil componentID, got %v", r.ComponentID)
	}
}

func TestRequirementResolution_Normalize_NilSafe(t *testing.T) {
	var r *domain.RequirementResolution
	r.Normalize() // should not panic
}

func TestRequirementResolution_IsZero(t *testing.T) {
	cases := []struct {
		name string
		r    domain.RequirementResolution
		want bool
	}{
		{"empty", domain.RequirementResolution{}, true},
		{"kind only", domain.RequirementResolution{Kind: domain.RequirementResolutionKindInternalComponent}, false},
		{"component only", domain.RequirementResolution{ComponentID: strPtr("x")}, false},
		{"full", domain.RequirementResolution{Kind: domain.RequirementResolutionKindInternalComponent, ComponentID: strPtr("x")}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.IsZero(); got != tc.want {
				t.Errorf("IsZero() = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- ProjectRequirement.NormalizeResolution ---

func TestNormalizeResolution_NilResolution_FallsBackToSelectedComponentID(t *testing.T) {
	req := domain.ProjectRequirement{
		ID:                  "req-1",
		SelectedComponentID: strPtr("cid-1"),
	}
	req.NormalizeResolution()

	if req.Resolution == nil {
		t.Fatal("expected resolution to be populated from selectedComponentId")
	}
	if req.Resolution.Kind != domain.RequirementResolutionKindInternalComponent {
		t.Errorf("expected internal_component, got %q", req.Resolution.Kind)
	}
	if req.Resolution.ComponentID == nil || *req.Resolution.ComponentID != "cid-1" {
		t.Error("expected resolution to reference cid-1")
	}
}

func TestNormalizeResolution_ExistingResolution_PreservedAndSynced(t *testing.T) {
	req := domain.ProjectRequirement{
		ID: "req-1",
		Resolution: &domain.RequirementResolution{
			Kind:        domain.RequirementResolutionKindInternalComponent,
			ComponentID: strPtr("cid-2"),
		},
	}
	req.NormalizeResolution()

	if req.SelectedComponentID == nil || *req.SelectedComponentID != "cid-2" {
		t.Error("expected selectedComponentId to be synced from resolution")
	}
}

func TestNormalizeResolution_BothNil_StaysNil(t *testing.T) {
	req := domain.ProjectRequirement{ID: "req-1"}
	req.NormalizeResolution()

	if req.Resolution != nil {
		t.Errorf("expected nil resolution, got %#v", req.Resolution)
	}
	if req.SelectedComponentID != nil {
		t.Errorf("expected nil selectedComponentId, got %v", req.SelectedComponentID)
	}
}

func TestNormalizeResolution_ZeroResolution_ClearsToNil(t *testing.T) {
	req := domain.ProjectRequirement{
		ID:         "req-1",
		Resolution: &domain.RequirementResolution{Kind: "", ComponentID: nil},
	}
	req.NormalizeResolution()

	// Zero resolution should be cleaned up to nil
	// Then fallback from SelectedComponentID (also nil) yields nil
	if req.Resolution != nil {
		t.Errorf("expected nil resolution after normalizing zero, got %#v", req.Resolution)
	}
}

// --- ProjectRequirement.ResolvedComponentID ---

func TestResolvedComponentID_FromResolution(t *testing.T) {
	req := domain.ProjectRequirement{
		ID: "req-1",
		Resolution: &domain.RequirementResolution{
			Kind:        domain.RequirementResolutionKindInternalComponent,
			ComponentID: strPtr("cid-1"),
		},
	}
	cid := req.ResolvedComponentID()
	if cid == nil || *cid != "cid-1" {
		t.Errorf("expected cid-1, got %v", cid)
	}
}

func TestResolvedComponentID_FallsBackToSelectedComponentID(t *testing.T) {
	req := domain.ProjectRequirement{
		ID:                  "req-1",
		SelectedComponentID: strPtr("cid-legacy"),
	}
	cid := req.ResolvedComponentID()
	if cid == nil || *cid != "cid-legacy" {
		t.Errorf("expected cid-legacy, got %v", cid)
	}
}

func TestResolvedComponentID_NilWhenBothEmpty(t *testing.T) {
	req := domain.ProjectRequirement{ID: "req-1"}
	if cid := req.ResolvedComponentID(); cid != nil {
		t.Errorf("expected nil, got %v", cid)
	}
}

func TestResolvedComponentID_SupplierPartKind_ReturnsNil(t *testing.T) {
	req := domain.ProjectRequirement{
		ID: "req-1",
		Resolution: &domain.RequirementResolution{
			Kind: domain.RequirementResolutionKindSupplierPart,
		},
	}
	cid := req.ResolvedComponentID()
	if cid != nil {
		t.Errorf("expected nil for supplier_part resolution, got %v", cid)
	}
}
