package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"trace/internal/domain"
	"trace/internal/kicad"
)

const kicadImportSourceType = "kicad"

func (s *Service) ListKiCadProjects(ctx context.Context, roots []string, query string) ([]kicad.ProjectCandidate, error) {
	if s.kicad == nil {
		return nil, fmt.Errorf("KiCad importer not configured")
	}
	return s.kicad.ListProjects(ctx, roots, query)
}

func (s *Service) PreviewKiCadImport(ctx context.Context, projectPath string) (kicad.ImportPreviewResponse, error) {
	if s.kicad == nil {
		return kicad.ImportPreviewResponse{}, fmt.Errorf("KiCad importer not configured")
	}
	return s.kicad.PreviewImport(ctx, projectPath)
}

func (s *Service) ImportKiCadProject(ctx context.Context, input kicad.ImportCommitInput) (domain.Project, error) {
	requirements, err := requirementsFromKiCadRows(input.Rows)
	if err != nil {
		return domain.Project{}, err
	}
	if len(requirements) == 0 {
		return domain.Project{}, fmt.Errorf("select at least one row to import")
	}

	sourceType := kicadImportSourceType
	sourcePath := strings.TrimSpace(input.SourceProjectPath)
	importedAt := time.Now().UTC()

	switch input.TargetMode {
	case kicad.ImportTargetModeNew:
		projectName := strings.TrimSpace(input.NewProjectName)
		if projectName == "" {
			return domain.Project{}, fmt.Errorf("project name required")
		}
		created, err := s.CreateProject(ctx, domain.Project{
			ID:               strings.TrimSpace(input.NewProjectID),
			Name:             projectName,
			Description:      strings.TrimSpace(input.NewProjectDescription),
			ImportSourceType: &sourceType,
			ImportSourcePath: &sourcePath,
			ImportedAt:       &importedAt,
			Requirements:     requirements,
		})
		if err != nil {
			return domain.Project{}, err
		}
		return created, nil
	case kicad.ImportTargetModeExisting:
		projectID := strings.TrimSpace(input.ExistingProjectID)
		if projectID == "" {
			return domain.Project{}, fmt.Errorf("existing project id required")
		}
		if err := s.AddProjectRequirements(ctx, projectID, requirements); err != nil {
			return domain.Project{}, err
		}
		if err := s.SetProjectImportMetadata(ctx, projectID, &sourceType, &sourcePath, &importedAt); err != nil {
			return domain.Project{}, err
		}
		return s.GetProject(ctx, projectID)
	default:
		return domain.Project{}, fmt.Errorf("unsupported import target mode %q", input.TargetMode)
	}
}

func requirementsFromKiCadRows(rows []kicad.ImportPreviewRow) ([]domain.ProjectRequirement, error) {
	requirements := make([]domain.ProjectRequirement, 0, len(rows))
	for _, row := range rows {
		if !row.Included {
			continue
		}
		name := strings.TrimSpace(row.Requirement.Name)
		if name == "" {
			return nil, fmt.Errorf("row %s: requirement name required", row.RowID)
		}
		if strings.TrimSpace(string(row.Requirement.Category)) == "" {
			return nil, fmt.Errorf("row %s: requirement category required", row.RowID)
		}
		if row.Requirement.Quantity <= 0 {
			return nil, fmt.Errorf("row %s: quantity must be greater than zero", row.RowID)
		}
		requirements = append(requirements, domain.ProjectRequirement{
			Name:                name,
			Category:            row.Requirement.Category,
			Quantity:            row.Requirement.Quantity,
			SelectedComponentID: row.Requirement.SelectedComponentID,
			Constraints:         cloneConstraints(row.Requirement.Constraints),
		})
	}
	return requirements, nil
}

func cloneConstraints(constraints []domain.RequirementConstraint) []domain.RequirementConstraint {
	if len(constraints) == 0 {
		return nil
	}
	cloned := make([]domain.RequirementConstraint, len(constraints))
	copy(cloned, constraints)
	return cloned
}
