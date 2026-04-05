package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sort"
	"time"

	"fmt"

	"componentmanager/internal/domain"
	"componentmanager/internal/domain/registry"
	"componentmanager/internal/kicad"
	"componentmanager/internal/kicadconfig"
	"componentmanager/internal/sourcing"
	"componentmanager/internal/supplierconfig"
)

type Service struct {
	components     domain.ComponentRepository
	projects       domain.ProjectRepository
	assets         domain.ComponentAssetRepository
	kicad          *kicad.Service
	kicadConfig    *kicadconfig.Manager
	sourcing       *sourcing.Service
	supplierConfig *supplierconfig.Manager
}

func New(components domain.ComponentRepository, projects domain.ProjectRepository, assets domain.ComponentAssetRepository, kicadServices ...*kicad.Service) *Service {
	var importer *kicad.Service
	if len(kicadServices) > 0 {
		importer = kicadServices[0]
	}
	return &Service{components: components, projects: projects, assets: assets, kicad: importer}
}

func (s *Service) SetSourcing(sourcingSvc *sourcing.Service) *Service {
	s.sourcing = sourcingSvc
	return s
}

func (s *Service) SetKiCadConfig(configSvc *kicadconfig.Manager) *Service {
	s.kicadConfig = configSvc
	return s
}

func (s *Service) SetSupplierConfig(configSvc *supplierconfig.Manager) *Service {
	s.supplierConfig = configSvc
	return s
}

func (s *Service) GetKiCadPreferences(ctx context.Context) (kicadconfig.Preferences, error) {
	if s.kicadConfig == nil {
		return kicadconfig.Preferences{}, fmt.Errorf("KiCad preferences not configured")
	}
	return s.kicadConfig.GetPreferences(ctx)
}

func (s *Service) SaveKiCadPreferences(ctx context.Context, input kicadconfig.UpdateInput) (kicadconfig.Preferences, error) {
	if s.kicadConfig == nil {
		return kicadconfig.Preferences{}, fmt.Errorf("KiCad preferences not configured")
	}
	return s.kicadConfig.SavePreferences(ctx, input)
}

func (s *Service) GetSupplierPreferences(ctx context.Context) (supplierconfig.Preferences, error) {
	if s.supplierConfig == nil {
		return supplierconfig.Preferences{}, fmt.Errorf("supplier preferences not configured")
	}
	return s.supplierConfig.GetPreferences(ctx)
}

func (s *Service) SaveSupplierPreferences(ctx context.Context, input supplierconfig.UpdateInput) (supplierconfig.Preferences, error) {
	if s.supplierConfig == nil {
		return supplierconfig.Preferences{}, fmt.Errorf("supplier preferences not configured")
	}
	return s.supplierConfig.SavePreferences(ctx, input)
}

func (s *Service) ClearSupplierSecret(ctx context.Context, provider, secret string) (supplierconfig.Preferences, error) {
	if s.supplierConfig == nil {
		return supplierconfig.Preferences{}, fmt.Errorf("supplier preferences not configured")
	}
	return s.supplierConfig.ClearSecret(ctx, provider, secret)
}

func (s *Service) UpsertAttributeDefinition(ctx context.Context, def domain.AttributeDefinition) error {
	return s.components.UpsertAttributeDefinition(ctx, def)
}

func (s *Service) SyncCanonicalAttributeDefinitions(ctx context.Context) error {
	for _, category := range registry.Categories() {
		for _, def := range registry.DefinitionsForCategory(category) {
			if err := s.components.UpsertAttributeDefinition(ctx, def); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) CreateComponent(ctx context.Context, component domain.Component) (domain.Component, error) {
	if component.ID == "" {
		component.ID = newID()
	}

	if err := registry.ValidateAttributes(component.Category, component.Attributes); err != nil {
		return domain.Component{}, err
	}

	return s.components.CreateComponent(ctx, component)
}

func (s *Service) GetComponent(ctx context.Context, id string) (domain.Component, error) {
	return s.components.GetComponent(ctx, id)
}

func (s *Service) UpdateComponentMetadata(ctx context.Context, component domain.Component) (domain.Component, error) {
	return s.components.UpdateComponentMetadata(ctx, component)
}

func (s *Service) ReplaceComponentAttributes(ctx context.Context, componentID string, attrs []domain.AttributeValue) error {
	c, err := s.components.GetComponent(ctx, componentID)
	if err != nil {
		return err
	}
	if err := registry.ValidateAttributes(c.Category, attrs); err != nil {
		return err
	}
	return s.components.ReplaceComponentAttributes(ctx, componentID, attrs)
}

func (s *Service) FindComponents(ctx context.Context, filter domain.ComponentFilter) ([]domain.Component, error) {
	return s.components.FindComponents(ctx, filter)
}

func (s *Service) DeleteComponent(ctx context.Context, id string) error {
	return s.components.DeleteComponent(ctx, id)
}

func (s *Service) UpdateComponentInventory(ctx context.Context, component domain.Component) (domain.Component, error) {
	if component.QuantityMode == "" {
		component.QuantityMode = domain.QuantityModeUnknown
	}
	if err := component.ValidateInventory(); err != nil {
		return domain.Component{}, err
	}
	return s.components.UpdateComponentInventory(ctx, component)
}

// AdjustComponentQuantity increments or decrements the component's quantity by delta.
// Only valid when quantity_mode is exact or approximate.
func (s *Service) AdjustComponentQuantity(ctx context.Context, id string, delta int) (domain.Component, error) {
	c, err := s.components.GetComponent(ctx, id)
	if err != nil {
		return domain.Component{}, err
	}
	if c.QuantityMode == domain.QuantityModeUnknown || c.QuantityMode == "" {
		return domain.Component{}, fmt.Errorf("cannot adjust quantity when quantity_mode is unknown")
	}
	current := 0
	if c.Quantity != nil {
		current = *c.Quantity
	}
	next := current + delta
	if next < 0 {
		next = 0
	}
	c.Quantity = &next
	return s.components.UpdateComponentInventory(ctx, c)
}

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
	return s.projects.SetRequirementResolution(ctx, requirementID, nil)
}

func (s *Service) PlanProject(ctx context.Context, projectID string) (domain.ProjectPlan, error) {
	project, err := s.projects.GetProject(ctx, projectID)
	if err != nil {
		return domain.ProjectPlan{}, err
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

		plan.Requirements = append(plan.Requirements, domain.RequirementPlan{
			Requirement:            requirement,
			MatchingOnHandQuantity: matchingOnHand,
			ShortfallQuantity:      shortfall,
			SelectedPart:           selectedPart,
			Matches:                matches,
		})
	}

	return plan, nil
}

func (s *Service) SourceRequirement(ctx context.Context, requirementID string) (sourcing.SourceResult, error) {
	if s.sourcing == nil && s.supplierConfig == nil {
		return sourcing.SourceResult{}, fmt.Errorf("supplier sourcing not configured")
	}

	requirement, err := s.projects.GetRequirement(ctx, requirementID)
	if err != nil {
		return sourcing.SourceResult{}, err
	}
	requirement.NormalizeResolution()

	var selectedDefinition *domain.Component
	if componentID := requirement.ResolvedComponentID(); componentID != nil {
		component, err := s.components.GetComponent(ctx, *componentID)
		if err != nil {
			return sourcing.SourceResult{}, err
		}
		selectedDefinition = &component
	}

	query := sourcing.BuildRequirementQuery(requirement, selectedDefinition)
	if s.supplierConfig != nil {
		sourcingSvc, err := s.supplierConfig.BuildSourcingService(ctx)
		if err != nil {
			return sourcing.SourceResult{}, err
		}
		return sourcingSvc.Source(ctx, query), nil
	}
	return s.sourcing.Source(ctx, query), nil
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

func componentDefinitionLabel(component domain.Component) string {
	parts := []string{component.Manufacturer, component.MPN}
	label := ""
	for _, part := range parts {
		part = fmt.Sprintf("%s", part)
		if part == "" {
			continue
		}
		if label == "" {
			label = part
			continue
		}
		label += " " + part
	}
	if label != "" {
		return label
	}
	if component.Description != "" {
		return component.Description
	}
	return component.ID
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

func newID() string {
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

// --- Component Assets ---

func (s *Service) CreateComponentAsset(ctx context.Context, asset domain.ComponentAsset) (domain.ComponentAsset, error) {
	if asset.ID == "" {
		asset.ID = newID()
	}
	if !asset.AssetType.Valid() {
		return domain.ComponentAsset{}, fmt.Errorf("invalid asset type %q", asset.AssetType)
	}
	if !asset.Status.Valid() {
		asset.Status = domain.AssetStatusCandidate
	}
	if asset.Source == "" {
		asset.Source = "manual"
	}
	// Verify the component exists.
	if _, err := s.components.GetComponent(ctx, asset.ComponentID); err != nil {
		return domain.ComponentAsset{}, err
	}
	return s.assets.CreateComponentAsset(ctx, asset)
}

func (s *Service) ListComponentAssets(ctx context.Context, componentID string) ([]domain.ComponentAsset, error) {
	return s.assets.ListComponentAssets(ctx, componentID)
}

func (s *Service) ListComponentAssetsByType(ctx context.Context, componentID string, assetType domain.AssetType) ([]domain.ComponentAsset, error) {
	if !assetType.Valid() {
		return nil, fmt.Errorf("invalid asset type %q", assetType)
	}
	return s.assets.ListComponentAssetsByType(ctx, componentID, assetType)
}

func (s *Service) GetComponentAsset(ctx context.Context, id string) (domain.ComponentAsset, error) {
	return s.assets.GetComponentAsset(ctx, id)
}

func (s *Service) UpdateComponentAsset(ctx context.Context, asset domain.ComponentAsset) (domain.ComponentAsset, error) {
	if asset.Status != "" && !asset.Status.Valid() {
		return domain.ComponentAsset{}, fmt.Errorf("invalid asset status %q", asset.Status)
	}
	return s.assets.UpdateComponentAsset(ctx, asset)
}

func (s *Service) DeleteComponentAsset(ctx context.Context, id string) error {
	return s.assets.DeleteComponentAsset(ctx, id)
}

func (s *Service) SetSelectedComponentAsset(ctx context.Context, componentID string, assetType domain.AssetType, assetID string) error {
	if !assetType.Valid() {
		return fmt.Errorf("invalid asset type %q", assetType)
	}
	return s.assets.SetSelectedComponentAsset(ctx, componentID, assetType, assetID)
}

func (s *Service) ClearSelectedComponentAsset(ctx context.Context, componentID string, assetType domain.AssetType) error {
	if !assetType.Valid() {
		return fmt.Errorf("invalid asset type %q", assetType)
	}
	return s.assets.ClearSelectedComponentAsset(ctx, componentID, assetType)
}

func (s *Service) GetComponentWithAssets(ctx context.Context, componentID string) (domain.ComponentWithAssets, error) {
	return s.assets.GetComponentWithAssets(ctx, componentID)
}

func (s *Service) UpdateComponentAssetStatus(ctx context.Context, assetID string, status domain.AssetStatus) (domain.ComponentAsset, error) {
	if !status.Valid() {
		return domain.ComponentAsset{}, fmt.Errorf("invalid asset status %q", status)
	}
	asset, err := s.assets.GetComponentAsset(ctx, assetID)
	if err != nil {
		return domain.ComponentAsset{}, err
	}
	asset.Status = status
	return s.assets.UpdateComponentAsset(ctx, asset)
}
