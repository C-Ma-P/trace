package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"componentmanager/internal/domain"
	"componentmanager/internal/kicad"
	"componentmanager/internal/paths"
)

func (a *App) ListKiCadProjects(roots []string, query string) ([]KiCadProjectCandidateResponse, error) {
	if err := a.checkReady(); err != nil {
		return nil, err
	}
	projects, err := a.svc.ListKiCadProjects(context.Background(), roots, query)
	if err != nil {
		return nil, err
	}
	out := make([]KiCadProjectCandidateResponse, len(projects))
	for i, project := range projects {
		out[i] = kiCadCandidateToResponse(project)
	}
	return out, nil
}

func (a *App) PreviewKiCadImport(projectPath string) (KiCadImportPreviewResponse, error) {
	if err := a.checkReady(); err != nil {
		return KiCadImportPreviewResponse{}, err
	}
	preview, err := a.svc.PreviewKiCadImport(context.Background(), projectPath)
	if err != nil {
		return KiCadImportPreviewResponse{}, err
	}
	return kiCadPreviewToResponse(preview), nil
}

func (a *App) ImportKiCadProject(input KiCadImportCommitInput) (ProjectResponse, error) {
	if err := a.checkReady(); err != nil {
		return ProjectResponse{}, err
	}

	newProjectID := ""
	cleanupProjectDir := func() {}
	if input.TargetMode == string(kicad.ImportTargetModeNew) {
		newProjectID = newID()
		projectDir, err := createProjectDiskState(newProjectID, input.NewProjectName, input.NewProjectDescription)
		if err != nil {
			return ProjectResponse{}, err
		}
		cleanupProjectDir = func() {
			_ = os.RemoveAll(projectDir)
		}
	}

	project, err := a.svc.ImportKiCadProject(context.Background(), kicad.ImportCommitInput{
		TargetMode:            kicad.ImportTargetMode(input.TargetMode),
		NewProjectID:          newProjectID,
		NewProjectName:        input.NewProjectName,
		NewProjectDescription: input.NewProjectDescription,
		ExistingProjectID:     input.ExistingProjectID,
		SourceProjectPath:     input.SourceProjectPath,
		Rows:                  responseRowsToPreviewRows(input.Rows),
	})
	if err != nil {
		cleanupProjectDir()
		return ProjectResponse{}, err
	}
	if a.launcher != nil {
		_ = a.launcher.TouchProject(project.ID, project.Name, project.Description)
	}
	return projectToResponse(project), nil
}

func createProjectDiskState(projectID, name, description string) (string, error) {
	projectsDir, err := paths.EnsureProjectsDir()
	if err != nil {
		return "", err
	}
	projectDir := filepath.Join(projectsDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return "", fmt.Errorf("create project dir: %w", err)
	}
	metadataPath := filepath.Join(projectDir, "project.json")
	metadataBytes, err := json.Marshal(struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		CreatedAt   string `json:"createdAt"`
	}{
		ID:          projectID,
		Name:        name,
		Description: description,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		_ = os.RemoveAll(projectDir)
		return "", err
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0o644); err != nil {
		_ = os.RemoveAll(projectDir)
		return "", fmt.Errorf("write project metadata: %w", err)
	}
	return projectDir, nil
}

func kiCadCandidateToResponse(candidate kicad.ProjectCandidate) KiCadProjectCandidateResponse {
	return KiCadProjectCandidateResponse{
		Name:        candidate.Name,
		ProjectPath: candidate.ProjectPath,
		ProjectDir:  candidate.ProjectDir,
	}
}

func kiCadPreviewToResponse(preview kicad.ImportPreviewResponse) KiCadImportPreviewResponse {
	rows := make([]KiCadImportPreviewRow, len(preview.Rows))
	for i, row := range preview.Rows {
		rows[i] = previewRowToResponse(row)
	}
	return KiCadImportPreviewResponse{
		SelectedProject: kiCadCandidateToResponse(preview.SelectedProject),
		Rows:            rows,
		Summary: KiCadImportPreviewSummary{
			TotalRows:    preview.Summary.TotalRows,
			IncludedRows: preview.Summary.IncludedRows,
			WarningRows:  preview.Summary.WarningRows,
		},
	}
}

func previewRowToResponse(row kicad.ImportPreviewRow) KiCadImportPreviewRow {
	constraints := make([]RequirementConstraintInput, len(row.Requirement.Constraints))
	for i, constraint := range row.Requirement.Constraints {
		constraints[i] = RequirementConstraintInput{
			Key:       constraint.Key,
			ValueType: string(constraint.ValueType),
			Operator:  string(constraint.Operator),
			Text:      constraint.Text,
			Number:    constraint.Number,
			Bool:      constraint.Bool,
			Unit:      constraint.Unit,
		}
	}
	return KiCadImportPreviewRow{
		RowID:          row.RowID,
		Included:       row.Included,
		SourceRefs:     row.SourceRefs,
		SourceQuantity: row.SourceQuantity,
		RawValue:       row.RawValue,
		RawFootprint:   row.RawFootprint,
		RawDescription: row.RawDescription,
		Manufacturer:   row.Manufacturer,
		MPN:            row.MPN,
		OtherFields:    row.OtherFields,
		Requirement: KiCadImportDraftRequirement{
			ID:                  "",
			ProjectID:           "",
			Name:                row.Requirement.Name,
			Category:            string(row.Requirement.Category),
			Quantity:            row.Requirement.Quantity,
			SelectedComponentID: row.Requirement.SelectedComponentID,
			Constraints:         constraints,
		},
		HasWarning:      row.HasWarning,
		WarningMessages: row.WarningMessages,
	}
}

func responseRowsToPreviewRows(rows []KiCadImportPreviewRow) []kicad.ImportPreviewRow {
	converted := make([]kicad.ImportPreviewRow, len(rows))
	for i, row := range rows {
		constraints := make([]domain.RequirementConstraint, len(row.Requirement.Constraints))
		for j, constraint := range row.Requirement.Constraints {
			constraints[j] = domain.RequirementConstraint{
				Key:       constraint.Key,
				ValueType: domain.ValueType(constraint.ValueType),
				Operator:  domain.Operator(constraint.Operator),
				Text:      constraint.Text,
				Number:    constraint.Number,
				Bool:      constraint.Bool,
				Unit:      constraint.Unit,
			}
		}
		converted[i] = kicad.ImportPreviewRow{
			RowID:          row.RowID,
			Included:       row.Included,
			SourceRefs:     row.SourceRefs,
			SourceQuantity: row.SourceQuantity,
			RawValue:       row.RawValue,
			RawFootprint:   row.RawFootprint,
			RawDescription: row.RawDescription,
			Manufacturer:   row.Manufacturer,
			MPN:            row.MPN,
			OtherFields:    row.OtherFields,
			Requirement: kicad.DraftRequirement{
				Name:                row.Requirement.Name,
				Category:            domain.Category(row.Requirement.Category),
				Quantity:            row.Requirement.Quantity,
				SelectedComponentID: row.Requirement.SelectedComponentID,
				Constraints:         constraints,
			},
			HasWarning:      row.HasWarning,
			WarningMessages: row.WarningMessages,
		}
	}
	return converted
}
