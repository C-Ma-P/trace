package kicad

import "github.com/C-Ma-P/trace/internal/domain"

type ProjectCandidate struct {
	Name        string
	ProjectPath string
	ProjectDir  string
}

type DraftRequirement struct {
	Name                string
	Category            domain.Category
	Quantity            int
	SelectedComponentID *string
	Constraints         []domain.RequirementConstraint
}

type ImportPreviewRow struct {
	RowID           string
	Included        bool
	SourceRefs      string
	SourceQuantity  int
	RawValue        string
	RawFootprint    string
	RawDescription  string
	Manufacturer    string
	MPN             string
	OtherFields     map[string]string
	Requirement     DraftRequirement
	HasWarning      bool
	WarningMessages []string
}

type ImportPreviewSummary struct {
	TotalRows    int
	IncludedRows int
	WarningRows  int
}

type ImportPreviewResponse struct {
	SelectedProject ProjectCandidate
	Rows            []ImportPreviewRow
	Summary         ImportPreviewSummary
}

type ImportTargetMode string

const (
	ImportTargetModeNew      ImportTargetMode = "new"
	ImportTargetModeExisting ImportTargetMode = "existing"
)

type ImportCommitInput struct {
	TargetMode            ImportTargetMode
	NewProjectID          string
	NewProjectName        string
	NewProjectDescription string
	ExistingProjectID     string
	SourceProjectPath     string
	Rows                  []ImportPreviewRow
}
