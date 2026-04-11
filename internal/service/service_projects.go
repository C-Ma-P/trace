package service

import (
	"context"
	"sort"
	"time"

	"trace/internal/domain"
	"trace/internal/domain/registry"
	"trace/internal/sourcing"
)

func (s *Service) CreateProject(ctx context.Context, project domain.Project) (domain.Project, error) {
	if project.ID == "" {
		project.ID = newID()
	}

	for i := range project.Requirements {
		if project.Requirements[i].ID == "" {
			project.Requirements[i].ID = newID()
		}
		project.Requirements[i].ProjectID = project.ID
		project.Requirements[i].NormalizeResolution()

		if err := registry.ValidateConstraints(project.Requirements[i].Category, project.Requirements[i].Constraints); err != nil {
			return domain.Project{}, err
		}
	}

	return s.projects.CreateProject(ctx, project)
}

func (s *Service) GetProject(ctx context.Context, id string) (domain.Project, error) {
	return s.projects.GetProject(ctx, id)
}

func (s *Service) ListProjects(ctx context.Context) ([]domain.Project, error) {
	return s.projects.ListProjects(ctx)
}

func (s *Service) DeleteProject(ctx context.Context, id string) error {
	return s.projects.DeleteProject(ctx, id)
}

func (s *Service) UpdateProject(ctx context.Context, project domain.Project) (domain.Project, error) {
	return s.projects.UpdateProject(ctx, project)
}

func (s *Service) ReplaceProjectRequirements(ctx context.Context, projectID string, requirements []domain.ProjectRequirement) error {
	requirements, err := s.prepareProjectRequirements(ctx, projectID, requirements)
	if err != nil {
		return err
	}
	return s.projects.ReplaceProjectRequirements(ctx, projectID, requirements)
}

func (s *Service) AddProjectRequirements(ctx context.Context, projectID string, requirements []domain.ProjectRequirement) error {
	requirements, err := s.prepareProjectRequirements(ctx, projectID, requirements)
	if err != nil {
		return err
	}
	return s.projects.AddProjectRequirements(ctx, projectID, requirements)
}

func (s *Service) SetProjectImportMetadata(ctx context.Context, projectID string, sourceType, sourcePath *string, importedAt *time.Time) error {
	return s.projects.SetProjectImportMetadata(ctx, projectID, sourceType, sourcePath, importedAt)
}

func (s *Service) prepareProjectRequirements(ctx context.Context, projectID string, requirements []domain.ProjectRequirement) ([]domain.ProjectRequirement, error) {
	for i := range requirements {
		if requirements[i].ID == "" {
			requirements[i].ID = newID()
		}
		requirements[i].ProjectID = projectID
		requirements[i].NormalizeResolution()

		if err := registry.ValidateConstraints(requirements[i].Category, requirements[i].Constraints); err != nil {
			return nil, err
		}

		if componentID := requirements[i].ResolvedComponentID(); componentID != nil {
			component, err := s.components.GetComponent(ctx, *componentID)
			if err != nil {
				return nil, err
			}
			if component.Category != requirements[i].Category {
				return nil, domain.ErrCategoryMismatch{
					RequirementCategory: requirements[i].Category,
					ComponentCategory:   component.Category,
				}
			}
			if _, ok := matchesRequirement(component, requirements[i]); !ok {
				return nil, domain.ErrRequirementNotSatisfied{
					ComponentID:   component.ID,
					RequirementID: requirements[i].ID,
				}
			}
		}
	}
	return requirements, nil
}

func (s *Service) ResolveRequirementToComponent(ctx context.Context, requirementID, componentID string) error {
	req, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return err
	}
	component, err := s.components.GetComponent(ctx, componentID)
	if err != nil {
		return err
	}
	if component.Category != req.Category {
		return domain.ErrCategoryMismatch{
			RequirementCategory: req.Category,
			ComponentCategory:   component.Category,
		}
	}
	if _, ok := matchesRequirement(component, req); !ok {
		return domain.ErrRequirementNotSatisfied{
			ComponentID:   componentID,
			RequirementID: requirementID,
		}
	}
	resolution := domain.NewComponentRequirementResolution(componentID)
	return s.projects.SetRequirementResolution(ctx, requirementID, resolution)
}

func (s *Service) SelectComponentForRequirement(ctx context.Context, requirementID, componentID string) error {
	return s.ResolveRequirementToComponent(ctx, requirementID, componentID)
}

func (s *Service) ClearSelectedComponentForRequirement(ctx context.Context, requirementID string) error {
	// Clear preferred candidate and requirement resolution together (invariant).
	if err := s.projects.ClearPreferredCandidate(ctx, requirementID); err != nil {
		return err
	}
	return s.projects.SetRequirementResolution(ctx, requirementID, nil)
}

func (s *Service) PlanProject(ctx context.Context, projectID string) (domain.ProjectPlan, error) {
	project, err := s.projects.GetProject(ctx, projectID)
	if err != nil {
		return domain.ProjectPlan{}, err
	}

	allCandidates, err := s.projects.ListPartCandidatesByProject(ctx, projectID)
	if err != nil {
		return domain.ProjectPlan{}, err
	}
	allCandidates, _ = s.hydrateCandidates(ctx, allCandidates)
	candidatesByReq := make(map[string][]domain.ProjectPartCandidate)
	for _, c := range allCandidates {
		candidatesByReq[c.RequirementID] = append(candidatesByReq[c.RequirementID], c)
	}

	allOffers, err := s.projects.ListSavedSupplierOffersByProject(ctx, projectID)
	if err != nil {
		return domain.ProjectPlan{}, err
	}
	offersByReq := make(map[string][]domain.SavedSupplierOffer)
	for _, o := range allOffers {
		offersByReq[o.RequirementID] = append(offersByReq[o.RequirementID], o)
	}

	plan := domain.ProjectPlan{Project: project}
	for _, requirement := range project.Requirements {
		requirement.NormalizeResolution()
		matches, err := s.MatchRequirement(ctx, requirement, true)
		if err != nil {
			return domain.ProjectPlan{}, err
		}

		matchingOnHand := 0
		for _, match := range matches {
			matchingOnHand += match.OnHandQuantity
		}
		shortfall := requirement.Quantity - matchingOnHand
		if shortfall < 0 {
			shortfall = 0
		}

		selectedPart, err := s.buildSelectedPart(ctx, requirement)
		if err != nil {
			return domain.ProjectPlan{}, err
		}

		candidates := candidatesByReq[requirement.ID]
		if candidates == nil {
			candidates = []domain.ProjectPartCandidate{}
		}
		offers := offersByReq[requirement.ID]
		if offers == nil {
			offers = []domain.SavedSupplierOffer{}
		}

		preferredOfferIDs := make(map[string]struct{})
		for _, c := range candidates {
			if c.Preferred && c.Origin == domain.CandidateOriginProvider && c.SourceOfferID != nil {
				preferredOfferIDs[*c.SourceOfferID] = struct{}{}
			}
		}
		for i := range offers {
			if _, ok := preferredOfferIDs[offers[i].ID]; !ok {
				continue
			}
			if offers[i].AssetProbeState == "" || offers[i].AssetProbeState == string(sourcing.AssetProbeStateUnknown) {
				s.maybeEnrichSavedSupplierOffer(ctx, &offers[i])
			}
		}

		readiness := s.computeExportReadiness(ctx, candidates)

		plan.Requirements = append(plan.Requirements, domain.RequirementPlan{
			Requirement:            requirement,
			MatchingOnHandQuantity: matchingOnHand,
			ShortfallQuantity:      shortfall,
			SelectedPart:           selectedPart,
			Matches:                matches,
			Candidates:             candidates,
			SavedOffers:            offers,
			Readiness:              readiness,
		})
	}

	return plan, nil
}

func (s *Service) MatchRequirement(ctx context.Context, requirement domain.ProjectRequirement, stockedOnly bool) ([]domain.ComponentMatch, error) {
	components, err := s.components.ListComponentsByCategory(ctx, requirement.Category)
	if err != nil {
		return nil, err
	}

	matches := make([]domain.ComponentMatch, 0)
	for _, component := range components {
		available := totalQuantity(component)
		if stockedOnly && available == 0 {
			continue
		}

		score, ok := matchesRequirement(component, requirement)
		if !ok {
			continue
		}

		matches = append(matches, domain.ComponentMatch{
			Component:      component,
			OnHandQuantity: available,
			Score:          score,
		})
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score != matches[j].Score {
			return matches[i].Score > matches[j].Score
		}
		if matches[i].OnHandQuantity != matches[j].OnHandQuantity {
			return matches[i].OnHandQuantity > matches[j].OnHandQuantity
		}
		return matches[i].Component.MPN < matches[j].Component.MPN
	})

	return matches, nil
}

func totalQuantity(c domain.Component) int {
	if c.QuantityMode != domain.QuantityModeUnknown && c.QuantityMode != "" && c.Quantity != nil {
		return *c.Quantity
	}
	return 0
}

func matchesRequirement(component domain.Component, requirement domain.ProjectRequirement) (int, bool) {
	index := component.AttributeIndex()

	score := 0
	for _, constraint := range requirement.Constraints {
		if matched, ok := matchesMetadataConstraint(component, constraint); ok {
			if !matched {
				return 0, false
			}
			score++
			continue
		}

		attribute, ok := index[constraint.Key]
		if !ok {
			return 0, false
		}

		if !valueMatches(attribute, constraint) {
			return 0, false
		}

		score++
	}

	if componentID := requirement.ResolvedComponentID(); componentID != nil && component.ID == *componentID {
		score += 1000
	}

	return score, true
}

func (s *Service) buildSelectedPart(ctx context.Context, requirement domain.ProjectRequirement) (*domain.RequirementSelectedPart, error) {
	resolution := requirement.Resolution
	if resolution == nil {
		return nil, nil
	}
	if resolution.Kind != domain.RequirementResolutionKindInternalComponent || resolution.ComponentID == nil {
		return &domain.RequirementSelectedPart{
			Resolution:  *resolution,
			DisplayName: "Resolved part",
		}, nil
	}
	component, err := s.components.GetComponent(ctx, *resolution.ComponentID)
	if err != nil {
		return nil, err
	}
	onHandQuantity := totalQuantity(component)
	shortfallQuantity := requirement.Quantity - onHandQuantity
	if shortfallQuantity < 0 {
		shortfallQuantity = 0
	}
	return &domain.RequirementSelectedPart{
		Resolution:        *resolution,
		DisplayName:       componentDefinitionLabel(component),
		Component:         &component,
		OnHandQuantity:    onHandQuantity,
		ShortfallQuantity: shortfallQuantity,
	}, nil
}

// computeExportReadiness determines the KiCad export readiness of a requirement
// based on its candidates and the preferred component's asset state.
func (s *Service) computeExportReadiness(ctx context.Context, candidates []domain.ProjectPartCandidate) domain.RequirementReadiness {
	var preferred *domain.ProjectPartCandidate
	for i := range candidates {
		if candidates[i].Preferred {
			preferred = &candidates[i]
			break
		}
	}

	if preferred == nil {
		return domain.RequirementReadiness{
			Status:   domain.ReadinessMissingPreferred,
			Blockers: []string{"No preferred component selected"},
		}
	}

	if preferred.Origin == domain.CandidateOriginProvider {
		return domain.RequirementReadiness{
			Status:   domain.ReadinessProviderBacked,
			Blockers: []string{"Preferred candidate is provider-backed — import into catalog first"},
		}
	}

	if preferred.ComponentID == nil {
		return domain.RequirementReadiness{
			Status:   domain.ReadinessMissingPreferred,
			Blockers: []string{"Preferred candidate has no linked component"},
		}
	}

	// Check component's selected assets.
	detail, err := s.assets.GetComponentWithAssets(ctx, *preferred.ComponentID)
	if err != nil {
		return domain.RequirementReadiness{
			Status:   domain.ReadinessMissingPreferred,
			Blockers: []string{"Cannot load component assets"},
		}
	}

	var blockers []string
	if detail.SelectedSymbolAsset == nil {
		blockers = append(blockers, "Missing selected symbol")
	}
	if detail.SelectedFootprintAsset == nil {
		blockers = append(blockers, "Missing selected footprint")
	}

	if len(blockers) > 0 {
		status := domain.ReadinessMissingFootprint
		if detail.SelectedSymbolAsset == nil {
			status = domain.ReadinessMissingSymbol
		}
		return domain.RequirementReadiness{Status: status, Blockers: blockers}
	}

	return domain.RequirementReadiness{
		Status:   domain.ReadinessReady,
		Blockers: []string{},
	}
}

func matchesMetadataConstraint(component domain.Component, constraint domain.RequirementConstraint) (bool, bool) {
	if constraint.ValueType != domain.ValueTypeText || constraint.Operator != domain.OperatorEqual || constraint.Text == nil {
		return false, false
	}

	actual := ""
	switch constraint.Key {
	case registry.AttrManufacturer:
		actual = component.Manufacturer
	case registry.AttrMPN:
		actual = component.MPN
	case registry.AttrPackage:
		actual = component.Package
	default:
		return false, false
	}

	return actual == *constraint.Text, true
}

func valueMatches(attribute domain.AttributeValue, constraint domain.RequirementConstraint) bool {
	if attribute.ValueType != constraint.ValueType {
		return false
	}

	switch constraint.ValueType {
	case domain.ValueTypeText:
		if attribute.Text == nil || constraint.Text == nil {
			return false
		}
		return constraint.Operator == domain.OperatorEqual && *attribute.Text == *constraint.Text
	case domain.ValueTypeBool:
		if attribute.Bool == nil || constraint.Bool == nil {
			return false
		}
		return constraint.Operator == domain.OperatorEqual && *attribute.Bool == *constraint.Bool
	case domain.ValueTypeNumber:
		if attribute.Number == nil || constraint.Number == nil {
			return false
		}
		switch constraint.Operator {
		case domain.OperatorEqual:
			return *attribute.Number == *constraint.Number
		case domain.OperatorGTE:
			return *attribute.Number >= *constraint.Number
		case domain.OperatorLTE:
			return *attribute.Number <= *constraint.Number
		default:
			return false
		}
	default:
		return false
	}
}
