package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"componentmanager/internal/domain"
	"componentmanager/internal/paths"
	"componentmanager/internal/sourcing"
)

func (a *App) ListProjects() ([]ProjectResponse, error) {
	if err := a.checkReady(); err != nil {
		return nil, err
	}
	projects, err := a.svc.ListProjects(context.Background())
	if err != nil {
		return nil, err
	}
	out := make([]ProjectResponse, len(projects))
	for i, p := range projects {
		out[i] = projectToResponse(p)
	}
	return out, nil
}

func (a *App) GetProject(id string) (ProjectResponse, error) {
	if err := a.checkReady(); err != nil {
		return ProjectResponse{}, err
	}
	p, err := a.svc.GetProject(context.Background(), id)
	if err != nil {
		return ProjectResponse{}, err
	}
	if a.launcher != nil {
		_ = a.launcher.TouchProject(p.ID, p.Name, p.Description)
	}
	return projectToResponse(p), nil
}

func (a *App) CreateProject(req CreateProjectInput) (ProjectResponse, error) {
	if err := a.checkReady(); err != nil {
		return ProjectResponse{}, err
	}
	p, err := a.svc.CreateProject(context.Background(), domain.Project{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return ProjectResponse{}, err
	}
	if a.launcher != nil {
		_ = a.launcher.TouchProject(p.ID, p.Name, p.Description)
	}
	return projectToResponse(p), nil
}

func (a *App) CreateBlankProject() (ProjectResponse, error) {
	if err := a.checkReady(); err != nil {
		return ProjectResponse{}, err
	}

	projectID := newID()
	projectName := "Untitled Project"
	projectDescription := ""

	projectsDir, err := paths.EnsureProjectsDir()
	if err != nil {
		return ProjectResponse{}, err
	}
	projectDir := filepath.Join(projectsDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return ProjectResponse{}, fmt.Errorf("create project dir: %w", err)
	}

	metadataPath := filepath.Join(projectDir, "project.json")
	metadataBytes, err := json.Marshal(struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		CreatedAt   string `json:"createdAt"`
	}{
		ID:          projectID,
		Name:        projectName,
		Description: projectDescription,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		_ = os.RemoveAll(projectDir)
		return ProjectResponse{}, err
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0o644); err != nil {
		_ = os.RemoveAll(projectDir)
		return ProjectResponse{}, fmt.Errorf("write project metadata: %w", err)
	}

	p, err := a.svc.CreateProject(context.Background(), domain.Project{
		ID:          projectID,
		Name:        projectName,
		Description: projectDescription,
	})
	if err != nil {
		_ = os.RemoveAll(projectDir)
		return ProjectResponse{}, err
	}
	if a.launcher != nil {
		_ = a.launcher.TouchProject(p.ID, p.Name, p.Description)
	}
	return projectToResponse(p), nil
}

func (a *App) CreateProjectWithDisk(req CreateProjectInput) (ProjectResponse, error) {
	if err := a.checkReady(); err != nil {
		return ProjectResponse{}, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return ProjectResponse{}, fmt.Errorf("project name required")
	}
	description := req.Description

	projectID := newID()

	projectsDir, err := paths.EnsureProjectsDir()
	if err != nil {
		return ProjectResponse{}, err
	}
	projectDir := filepath.Join(projectsDir, projectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return ProjectResponse{}, fmt.Errorf("create project dir: %w", err)
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
		return ProjectResponse{}, err
	}
	if err := os.WriteFile(metadataPath, metadataBytes, 0o644); err != nil {
		_ = os.RemoveAll(projectDir)
		return ProjectResponse{}, fmt.Errorf("write project metadata: %w", err)
	}

	p, err := a.svc.CreateProject(context.Background(), domain.Project{
		ID:          projectID,
		Name:        name,
		Description: description,
	})
	if err != nil {
		_ = os.RemoveAll(projectDir)
		return ProjectResponse{}, err
	}
	if a.launcher != nil {
		_ = a.launcher.TouchProject(p.ID, p.Name, p.Description)
	}
	return projectToResponse(p), nil
}

func (a *App) UpdateProjectMetadata(req UpdateProjectInput) (ProjectResponse, error) {
	if err := a.checkReady(); err != nil {
		return ProjectResponse{}, err
	}
	p, err := a.svc.UpdateProject(context.Background(), domain.Project{
		ID:          req.ID,
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return ProjectResponse{}, err
	}
	return projectToResponse(p), nil
}

func (a *App) DeleteProject(id string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.DeleteProject(context.Background(), id)
}

func (a *App) GetProjectDiskPath(projectID string) (string, error) {
	projectID = filepath.Clean(projectID)
	if projectID == "." || projectID == "" || projectID == string(filepath.Separator) {
		return "", fmt.Errorf("invalid project id")
	}
	projectsDir, err := paths.EnsureProjectsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(projectsDir, projectID), nil
}

func (a *App) RevealProjectInFileBrowser(projectID string) error {
	p, err := a.GetProjectDiskPath(projectID)
	if err != nil {
		return err
	}
	if _, err := os.Stat(p); err != nil {
		return err
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", p)
	case "windows":
		cmd = exec.Command("explorer.exe", p)
	default:
		cmd = exec.Command("xdg-open", p)
	}
	return cmd.Start()
}

func (a *App) DeleteProjectAndDisk(projectID string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	p, err := a.GetProjectDiskPath(projectID)
	if err != nil {
		return err
	}

	dbErr := a.svc.DeleteProject(context.Background(), projectID)
	var notFound domain.ErrNotFound
	if dbErr != nil && !errors.As(dbErr, &notFound) {
		return dbErr
	}

	if err := os.RemoveAll(p); err != nil {
		return fmt.Errorf("delete project dir: %w", err)
	}
	if a.launcher != nil {
		_ = a.launcher.RemoveProject(projectID)
	}
	return nil
}

func (a *App) ReplaceProjectRequirements(projectID string, reqs []RequirementInput) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	domainReqs := make([]domain.ProjectRequirement, len(reqs))
	for i, r := range reqs {
		constraints := make([]domain.RequirementConstraint, len(r.Constraints))
		for j, c := range r.Constraints {
			constraints[j] = domain.RequirementConstraint{
				Key:       c.Key,
				ValueType: domain.ValueType(c.ValueType),
				Operator:  domain.Operator(c.Operator),
				Text:      c.Text,
				Number:    c.Number,
				Bool:      c.Bool,
				Unit:      c.Unit,
			}
		}
		domainReqs[i] = requirementInputToDomain(r, constraints)
	}
	return a.svc.ReplaceProjectRequirements(context.Background(), projectID, domainReqs)
}

func (a *App) PlanProject(projectID string) (ProjectPlanResponse, error) {
	if err := a.checkReady(); err != nil {
		return ProjectPlanResponse{}, err
	}
	plan, err := a.svc.PlanProject(context.Background(), projectID)
	if err != nil {
		return ProjectPlanResponse{}, err
	}
	return planToResponse(plan), nil
}

func (a *App) SourceRequirement(requirementID string) (SourceRequirementResponse, error) {
	if err := a.checkReady(); err != nil {
		return SourceRequirementResponse{}, err
	}
	result, err := a.svc.SourceRequirement(context.Background(), requirementID)
	if err != nil {
		return SourceRequirementResponse{}, err
	}
	return sourceResultToResponse(result), nil
}

func (a *App) SelectComponentForRequirement(requirementID, componentID string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.SelectComponentForRequirement(context.Background(), requirementID, componentID)
}

func (a *App) ClearSelectedComponentForRequirement(requirementID string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.ClearSelectedComponentForRequirement(context.Background(), requirementID)
}

func projectToResponse(p domain.Project) ProjectResponse {
	reqs := make([]RequirementResponse, len(p.Requirements))
	for i, r := range p.Requirements {
		reqs[i] = requirementToResponse(r)
	}
	var importedAt *string
	if p.ImportedAt != nil {
		formatted := p.ImportedAt.Format(time.RFC3339)
		importedAt = &formatted
	}
	return ProjectResponse{
		ID:               p.ID,
		Name:             p.Name,
		Description:      p.Description,
		ImportSourceType: p.ImportSourceType,
		ImportSourcePath: p.ImportSourcePath,
		ImportedAt:       importedAt,
		Requirements:     reqs,
		CreatedAt:        p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        p.UpdatedAt.Format(time.RFC3339),
	}
}

func requirementToResponse(r domain.ProjectRequirement) RequirementResponse {
	r.NormalizeResolution()
	constraints := make([]ConstraintResponse, len(r.Constraints))
	for i, c := range r.Constraints {
		constraints[i] = ConstraintResponse{
			Key:       c.Key,
			ValueType: string(c.ValueType),
			Operator:  string(c.Operator),
			Text:      c.Text,
			Number:    c.Number,
			Bool:      c.Bool,
			Unit:      c.Unit,
		}
	}
	return RequirementResponse{
		ID:                  r.ID,
		ProjectID:           r.ProjectID,
		Name:                r.Name,
		Category:            string(r.Category),
		Quantity:            r.Quantity,
		SelectedComponentID: r.SelectedComponentID,
		Resolution:          requirementResolutionToResponse(r.Resolution),
		Constraints:         constraints,
	}
}

func planToResponse(plan domain.ProjectPlan) ProjectPlanResponse {
	reqs := make([]RequirementPlanResponse, len(plan.Requirements))
	for i, rp := range plan.Requirements {
		matches := make([]ComponentMatchResponse, len(rp.Matches))
		for j, m := range rp.Matches {
			matches[j] = ComponentMatchResponse{
				Component:      componentToResponse(m.Component),
				OnHandQuantity: m.OnHandQuantity,
				Score:          m.Score,
			}
		}
		candidates := make([]PartCandidateResponse, len(rp.Candidates))
		for j, c := range rp.Candidates {
			candidates[j] = partCandidateToResponse(c)
		}
		savedOffers := make([]SavedSupplierOfferResponse, len(rp.SavedOffers))
		for j, o := range rp.SavedOffers {
			savedOffers[j] = savedOfferToResponse(o)
		}
		reqs[i] = RequirementPlanResponse{
			Requirement:            requirementToResponse(rp.Requirement),
			MatchingOnHandQuantity: rp.MatchingOnHandQuantity,
			ShortfallQuantity:      rp.ShortfallQuantity,
			SelectedPart:           selectedPartToResponse(rp.SelectedPart),
			Matches:                matches,
			Candidates:             candidates,
			SavedOffers:            savedOffers,
		}
	}
	return ProjectPlanResponse{
		Project:      projectToResponse(plan.Project),
		Requirements: reqs,
	}
}

func requirementInputToDomain(input RequirementInput, constraints []domain.RequirementConstraint) domain.ProjectRequirement {
	requirement := domain.ProjectRequirement{
		ID:                  input.ID,
		Name:                input.Name,
		Category:            domain.Category(input.Category),
		Quantity:            input.Quantity,
		SelectedComponentID: input.SelectedComponentID,
		Constraints:         constraints,
	}
	if input.Resolution != nil {
		requirement.Resolution = &domain.RequirementResolution{
			Kind:        domain.RequirementResolutionKind(input.Resolution.Kind),
			ComponentID: input.Resolution.ComponentID,
		}
	}
	requirement.NormalizeResolution()
	return requirement
}

func requirementResolutionToResponse(resolution *domain.RequirementResolution) *RequirementResolutionResponse {
	if resolution == nil {
		return nil
	}
	copyResolution := *resolution
	copyResolution.Normalize()
	if copyResolution.IsZero() {
		return nil
	}
	return &RequirementResolutionResponse{
		Kind:        string(copyResolution.Kind),
		ComponentID: copyResolution.ComponentID,
	}
}

func selectedPartToResponse(selected *domain.RequirementSelectedPart) *RequirementSelectedPartResponse {
	if selected == nil {
		return nil
	}
	var component *ComponentResponse
	if selected.Component != nil {
		converted := componentToResponse(*selected.Component)
		component = &converted
	}
	resolution := requirementResolutionToResponse(&selected.Resolution)
	if resolution == nil {
		return nil
	}
	return &RequirementSelectedPartResponse{
		Resolution:        *resolution,
		DisplayName:       selected.DisplayName,
		Component:         component,
		OnHandQuantity:    selected.OnHandQuantity,
		ShortfallQuantity: selected.ShortfallQuantity,
	}
}

func sourceResultToResponse(result sourcing.SourceResult) SourceRequirementResponse {
	offers := make([]SupplierOfferResponse, len(result.Offers))
	for i, offer := range result.Offers {
		offers[i] = SupplierOfferResponse{
			Provider:           offer.Provider,
			Manufacturer:       offer.Manufacturer,
			MPN:                offer.MPN,
			SupplierPartNumber: offer.SupplierPartNumber,
			Description:        offer.Description,
			Package:            offer.Package,
			Stock:              offer.Stock,
			MOQ:                offer.MOQ,
			UnitPrice:          offer.UnitPrice,
			ProductURL:         offer.ProductURL,
			DatasheetURL:       offer.DatasheetURL,
			Lifecycle:          offer.Lifecycle,
			MatchScore:         offer.MatchScore,
			MatchReasons:       offer.MatchReasons,
			Raw:                offer.Raw,
		}
	}

	providers := make([]SupplierProviderStatusResponse, len(result.Providers))
	for i, provider := range result.Providers {
		providers[i] = SupplierProviderStatusResponse{
			Provider:   provider.Provider,
			Status:     provider.Status,
			Error:      provider.Error,
			OfferCount: provider.OfferCount,
		}
	}

	return SourceRequirementResponse{Offers: offers, Providers: providers, Currency: result.Currency}
}

// --- Part Candidates ---

func (a *App) AddPartCandidate(requirementID, componentID string) (PartCandidateResponse, error) {
	if err := a.checkReady(); err != nil {
		return PartCandidateResponse{}, err
	}
	candidate, err := a.svc.AddPartCandidate(context.Background(), requirementID, componentID)
	if err != nil {
		return PartCandidateResponse{}, err
	}
	return partCandidateToResponse(candidate), nil
}

func (a *App) SetPreferredCandidate(requirementID, candidateID string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.SetPreferredCandidate(context.Background(), requirementID, candidateID)
}

func (a *App) SetPreferredLocalComponent(requirementID, componentID string) (PartCandidateResponse, error) {
	if err := a.checkReady(); err != nil {
		return PartCandidateResponse{}, err
	}
	candidate, err := a.svc.AddLocalComponentAsCandidateAndSetPreferred(context.Background(), requirementID, componentID)
	if err != nil {
		return PartCandidateResponse{}, err
	}
	return partCandidateToResponse(candidate), nil
}

func (a *App) RemovePartCandidate(candidateID string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.RemovePartCandidate(context.Background(), candidateID)
}

// --- Saved Supplier Offers ---

func (a *App) SaveSupplierOffer(input SaveSupplierOfferInput) (SavedSupplierOfferResponse, error) {
	if err := a.checkReady(); err != nil {
		return SavedSupplierOfferResponse{}, err
	}
	offer := domain.SavedSupplierOffer{
		Provider:       input.Provider,
		ProviderPartID: input.ProviderPartID,
		ProductURL:     input.ProductURL,
		Manufacturer:   input.Manufacturer,
		MPN:            input.MPN,
		Description:    input.Description,
		Package:        input.Package,
		Stock:          input.Stock,
		MOQ:            input.MOQ,
		UnitPrice:      input.UnitPrice,
		Currency:       input.Currency,
		CapturedAt:     time.Now().UTC(),
	}
	saved, err := a.svc.SaveSupplierOfferForRequirement(context.Background(), input.RequirementID, offer)
	if err != nil {
		return SavedSupplierOfferResponse{}, err
	}
	return savedOfferToResponse(saved), nil
}

func (a *App) ImportSupplierOffer(input ImportSupplierOfferInput) (ImportSupplierOfferResponse, error) {
	if err := a.checkReady(); err != nil {
		return ImportSupplierOfferResponse{}, err
	}
	offer := domain.SavedSupplierOffer{
		Provider:       input.Provider,
		ProviderPartID: input.ProviderPartID,
		ProductURL:     input.ProductURL,
		Manufacturer:   input.Manufacturer,
		MPN:            input.MPN,
		Description:    input.Description,
		Package:        input.Package,
		Stock:          input.Stock,
		MOQ:            input.MOQ,
		UnitPrice:      input.UnitPrice,
		Currency:       input.Currency,
		CapturedAt:     time.Now().UTC(),
	}
	candidate, savedOffer, err := a.svc.ImportSupplierOffer(context.Background(), input.RequirementID, offer, input.SetPreferred)
	if err != nil {
		return ImportSupplierOfferResponse{}, err
	}
	return ImportSupplierOfferResponse{
		Candidate:  partCandidateToResponse(candidate),
		SavedOffer: savedOfferToResponse(savedOffer),
	}, nil
}

func (a *App) RemoveSavedSupplierOffer(offerID string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.RemoveSavedSupplierOffer(context.Background(), offerID)
}

func partCandidateToResponse(c domain.ProjectPartCandidate) PartCandidateResponse {
	var comp *ComponentResponse
	if c.Component != nil {
		cr := componentToResponse(*c.Component)
		comp = &cr
	}
	return PartCandidateResponse{
		ID:            c.ID,
		ProjectID:     c.ProjectID,
		RequirementID: c.RequirementID,
		ComponentID:   c.ComponentID,
		Preferred:     c.Preferred,
		Origin:        string(c.Origin),
		Component:     comp,
		CreatedAt:     c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     c.UpdatedAt.Format(time.RFC3339),
	}
}

func savedOfferToResponse(o domain.SavedSupplierOffer) SavedSupplierOfferResponse {
	return SavedSupplierOfferResponse{
		ID:                o.ID,
		ProjectID:         o.ProjectID,
		RequirementID:     o.RequirementID,
		Provider:          o.Provider,
		ProviderPartID:    o.ProviderPartID,
		ProductURL:        o.ProductURL,
		Manufacturer:      o.Manufacturer,
		MPN:               o.MPN,
		Description:       o.Description,
		Package:           o.Package,
		Stock:             o.Stock,
		MOQ:               o.MOQ,
		UnitPrice:         o.UnitPrice,
		Currency:          o.Currency,
		LinkedComponentID: o.LinkedComponentID,
		CapturedAt:        o.CapturedAt.Format(time.RFC3339),
		CreatedAt:         o.CreatedAt.Format(time.RFC3339),
	}
}
