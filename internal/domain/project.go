package domain

import (
	"strings"
	"time"
)

type RequirementResolutionKind string

const (
	RequirementResolutionKindInternalComponent RequirementResolutionKind = "internal_component"
	RequirementResolutionKindSupplierPart      RequirementResolutionKind = "supplier_part"
)

type RequirementResolution struct {
	Kind        RequirementResolutionKind `json:"kind"`
	ComponentID *string                   `json:"componentId,omitempty"`
}

func NewComponentRequirementResolution(componentID string) *RequirementResolution {
	componentID = strings.TrimSpace(componentID)
	if componentID == "" {
		return nil
	}
	return &RequirementResolution{
		Kind:        RequirementResolutionKindInternalComponent,
		ComponentID: &componentID,
	}
}

func (r *RequirementResolution) Normalize() {
	if r == nil {
		return
	}
	r.Kind = RequirementResolutionKind(strings.TrimSpace(string(r.Kind)))
	if r.ComponentID != nil {
		componentID := strings.TrimSpace(*r.ComponentID)
		if componentID == "" {
			r.ComponentID = nil
		} else {
			r.ComponentID = &componentID
		}
	}
	if r.Kind == RequirementResolutionKindInternalComponent && r.ComponentID == nil {
		r.Kind = ""
	}
}

func (r RequirementResolution) IsZero() bool {
	return r.Kind == "" && r.ComponentID == nil
}

type Project struct {
	ID               string               `db:"id"`
	Name             string               `db:"name"`
	Description      string               `db:"description"`
	ImportSourceType *string              `db:"import_source_type"`
	ImportSourcePath *string              `db:"import_source_path"`
	ImportedAt       *time.Time           `db:"imported_at"`
	Requirements     []ProjectRequirement `db:"-"`
	CreatedAt        time.Time            `db:"created_at"`
	UpdatedAt        time.Time            `db:"updated_at"`
}

type ProjectRequirement struct {
	ID                  string                  `db:"id"`
	ProjectID           string                  `db:"project_id"`
	Name                string                  `db:"name"`
	Category            Category                `db:"category"`
	Quantity            int                     `db:"quantity"`
	SelectedComponentID *string                 `db:"selected_component_id"`
	Resolution          *RequirementResolution  `db:"-"`
	Constraints         []RequirementConstraint `db:"-"`
}

func (r *ProjectRequirement) NormalizeResolution() {
	if r == nil {
		return
	}
	if r.Resolution != nil {
		r.Resolution.Normalize()
		if r.Resolution.IsZero() {
			r.Resolution = nil
		}
	}
	if r.Resolution == nil {
		r.Resolution = NewComponentRequirementResolution(derefString(r.SelectedComponentID))
	}
	if componentID := r.ResolvedComponentID(); componentID != nil {
		value := strings.TrimSpace(*componentID)
		r.SelectedComponentID = &value
		return
	}
	r.SelectedComponentID = nil
}

func (r ProjectRequirement) ResolvedComponentID() *string {
	if r.Resolution != nil && r.Resolution.Kind == RequirementResolutionKindInternalComponent && r.Resolution.ComponentID != nil {
		componentID := strings.TrimSpace(*r.Resolution.ComponentID)
		if componentID != "" {
			return &componentID
		}
	}
	if r.SelectedComponentID == nil {
		return nil
	}
	componentID := strings.TrimSpace(*r.SelectedComponentID)
	if componentID == "" {
		return nil
	}
	return &componentID
}

type RequirementConstraint struct {
	Key       string
	ValueType ValueType
	Operator  Operator
	Text      *string
	Number    *float64
	Bool      *bool
	Unit      string
}

type RequirementPlan struct {
	Requirement            ProjectRequirement
	MatchingOnHandQuantity int
	ShortfallQuantity      int
	SelectedPart           *RequirementSelectedPart
	Matches                []ComponentMatch
	Candidates             []ProjectPartCandidate
	SavedOffers            []SavedSupplierOffer
	Readiness              RequirementReadiness
}

type ProjectPlan struct {
	Project      Project
	Requirements []RequirementPlan
}

type RequirementSelectedPart struct {
	Resolution        RequirementResolution
	DisplayName       string
	Component         *Component
	OnHandQuantity    int
	ShortfallQuantity int
}

type CandidateOrigin string

const (
	CandidateOriginLocal            CandidateOrigin = "local"
	CandidateOriginImportedSupplier CandidateOrigin = "imported_from_supplier"
	CandidateOriginProvider         CandidateOrigin = "provider"
)

type ProjectPartCandidate struct {
	ID            string          `db:"id"`
	ProjectID     string          `db:"project_id"`
	RequirementID string          `db:"requirement_id"`
	ComponentID   *string         `db:"component_id"`
	SourceOfferID *string         `db:"source_offer_id"`
	Preferred     bool            `db:"preferred"`
	Origin        CandidateOrigin `db:"origin"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`

	// Hydrated by service layer, not persisted.
	Component   *Component          `db:"-"`
	SourceOffer *SavedSupplierOffer `db:"-"`
}

type SavedSupplierOffer struct {
	ID                string    `db:"id"`
	ProjectID         string    `db:"project_id"`
	RequirementID     string    `db:"requirement_id"`
	Provider          string    `db:"provider"`
	ProviderPartID    string    `db:"provider_part_id"`
	ProductURL        string    `db:"product_url"`
	ImageURL          string    `db:"image_url"`
	Manufacturer      string    `db:"manufacturer"`
	MPN               string    `db:"mpn"`
	Description       string    `db:"description"`
	Package           string    `db:"package"`
	Stock             *int      `db:"stock"`
	MOQ               *int      `db:"moq"`
	UnitPrice         *float64  `db:"unit_price"`
	Currency          string    `db:"currency"`
	LinkedComponentID *string   `db:"linked_component_id"`
	CapturedAt        time.Time `db:"captured_at"`
	CreatedAt         time.Time `db:"created_at"`
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// ExportReadiness describes the KiCad-export readiness status of a single
// requirement/component choice. Statuses are ordered from most-blocking to
// least-blocking so that the UI can use the "worst" status directly.
type ExportReadiness string

const (
	ReadinessReady            ExportReadiness = "ready"
	ReadinessMissingPreferred ExportReadiness = "missing_preferred"
	ReadinessProviderBacked   ExportReadiness = "provider_not_imported"
	ReadinessMissingSymbol    ExportReadiness = "missing_symbol"
	ReadinessMissingFootprint ExportReadiness = "missing_footprint"
)

// RequirementReadiness is the computed export-readiness state for a single
// requirement plan entry. Status is the worst blocker; Blockers lists all
// human-readable issues preventing readiness.
type RequirementReadiness struct {
	Status   ExportReadiness `json:"status"`
	Blockers []string        `json:"blockers"`
}
