package service_test

import (
	"context"
	"time"

	"github.com/C-Ma-P/trace/internal/domain"
)

type stubComponentRepo struct {
	upserted   []domain.AttributeDefinition
	getResult  domain.Component
	getErr     error
	findResult []domain.Component
	lastFilter domain.ComponentFilter

	updatedComp   *domain.Component
	updateCompErr error
	replacedID    string
	replacedAttrs []domain.AttributeValue
}

func (s *stubComponentRepo) CreateComponent(_ context.Context, c domain.Component) (domain.Component, error) {
	return c, nil
}
func (s *stubComponentRepo) GetComponent(_ context.Context, _ string) (domain.Component, error) {
	if s.getErr != nil {
		return domain.Component{}, s.getErr
	}
	return s.getResult, nil
}
func (s *stubComponentRepo) ListComponentsByCategory(_ context.Context, _ domain.Category) ([]domain.Component, error) {
	return nil, nil
}
func (s *stubComponentRepo) UpsertAttributeDefinition(_ context.Context, def domain.AttributeDefinition) error {
	s.upserted = append(s.upserted, def)
	return nil
}
func (s *stubComponentRepo) UpdateComponentMetadata(_ context.Context, c domain.Component) (domain.Component, error) {
	if s.updateCompErr != nil {
		return domain.Component{}, s.updateCompErr
	}
	s.updatedComp = &c
	return c, nil
}
func (s *stubComponentRepo) ReplaceComponentAttributes(_ context.Context, id string, attrs []domain.AttributeValue) error {
	s.replacedID = id
	s.replacedAttrs = attrs
	return nil
}
func (s *stubComponentRepo) FindComponents(_ context.Context, f domain.ComponentFilter) ([]domain.Component, error) {
	s.lastFilter = f
	return s.findResult, nil
}
func (s *stubComponentRepo) UpdateComponentInventory(_ context.Context, c domain.Component) (domain.Component, error) {
	return c, nil
}
func (s *stubComponentRepo) DeleteComponent(_ context.Context, _ string) error {
	return nil
}

type stubProjectRepo struct {
	created *domain.Project

	getResult          domain.Project
	getProjectErr      error
	listed             []domain.Project
	updatedProj        *domain.Project
	updateProjectErr   error
	replacedReqs       []domain.ProjectRequirement
	replacedForProject string
	addedReqs          []domain.ProjectRequirement
	addedForProject    string
	importMetaProject  string
	importSourceType   *string
	importSourcePath   *string
	importedAt         *time.Time

	getRequirementResult domain.ProjectRequirement
	getRequirementErr    error
	resolvedReqID        string
	resolution           *domain.RequirementResolution
}

func (s *stubProjectRepo) CreateProject(_ context.Context, p domain.Project) (domain.Project, error) {
	s.created = &p
	return p, nil
}
func (s *stubProjectRepo) GetProject(_ context.Context, _ string) (domain.Project, error) {
	if s.getProjectErr != nil {
		return domain.Project{}, s.getProjectErr
	}
	return s.getResult, nil
}
func (s *stubProjectRepo) ListProjects(_ context.Context) ([]domain.Project, error) {
	return s.listed, nil
}
func (s *stubProjectRepo) UpdateProject(_ context.Context, p domain.Project) (domain.Project, error) {
	if s.updateProjectErr != nil {
		return domain.Project{}, s.updateProjectErr
	}
	s.updatedProj = &p
	return p, nil
}
func (s *stubProjectRepo) DeleteProject(_ context.Context, _ string) error {
	return nil
}
func (s *stubProjectRepo) ReplaceProjectRequirements(_ context.Context, projectID string, reqs []domain.ProjectRequirement) error {
	s.replacedForProject = projectID
	s.replacedReqs = reqs
	return nil
}
func (s *stubProjectRepo) AddProjectRequirements(_ context.Context, projectID string, reqs []domain.ProjectRequirement) error {
	s.addedForProject = projectID
	s.addedReqs = reqs
	return nil
}
func (s *stubProjectRepo) SetProjectImportMetadata(_ context.Context, projectID string, sourceType, sourcePath *string, importedAt *time.Time) error {
	s.importMetaProject = projectID
	s.importSourceType = sourceType
	s.importSourcePath = sourcePath
	s.importedAt = importedAt
	return nil
}
func (s *stubProjectRepo) GetRequirement(_ context.Context, _ string) (domain.ProjectRequirement, error) {
	if s.getRequirementErr != nil {
		return domain.ProjectRequirement{}, s.getRequirementErr
	}
	return s.getRequirementResult, nil
}

func (s *stubProjectRepo) SetRequirementResolution(_ context.Context, reqID string, resolution *domain.RequirementResolution) error {
	s.resolvedReqID = reqID
	if resolution == nil {
		s.resolution = nil
		return nil
	}
	copyResolution := *resolution
	if resolution.ComponentID != nil {
		componentID := *resolution.ComponentID
		copyResolution.ComponentID = &componentID
	}
	s.resolution = &copyResolution
	return nil
}

func (s *stubProjectRepo) AddPartCandidate(_ context.Context, c domain.ProjectPartCandidate) (domain.ProjectPartCandidate, error) {
	return c, nil
}
func (s *stubProjectRepo) SetPreferredCandidate(_ context.Context, _, _ string) error {
	return nil
}
func (s *stubProjectRepo) ClearPreferredCandidate(_ context.Context, _ string) error {
	return nil
}
func (s *stubProjectRepo) GetPartCandidate(_ context.Context, id string) (domain.ProjectPartCandidate, error) {
	return domain.ProjectPartCandidate{ID: id}, nil
}
func (s *stubProjectRepo) RemovePartCandidate(_ context.Context, _ string) error {
	return nil
}
func (s *stubProjectRepo) ListPartCandidates(_ context.Context, _ string) ([]domain.ProjectPartCandidate, error) {
	return nil, nil
}
func (s *stubProjectRepo) ListPartCandidatesByProject(_ context.Context, _ string) ([]domain.ProjectPartCandidate, error) {
	return nil, nil
}
func (s *stubProjectRepo) SaveSupplierOffer(_ context.Context, o domain.SavedSupplierOffer) (domain.SavedSupplierOffer, error) {
	return o, nil
}
func (s *stubProjectRepo) RemoveSavedSupplierOffer(_ context.Context, _ string) error {
	return nil
}
func (s *stubProjectRepo) ListSavedSupplierOffers(_ context.Context, _ string) ([]domain.SavedSupplierOffer, error) {
	return nil, nil
}
func (s *stubProjectRepo) ListSavedSupplierOffersByProject(_ context.Context, _ string) ([]domain.SavedSupplierOffer, error) {
	return nil, nil
}
func (s *stubProjectRepo) LinkSupplierOfferToComponent(_ context.Context, _, _ string) error {
	return nil
}
func (s *stubProjectRepo) GetSavedSupplierOffer(_ context.Context, id string) (domain.SavedSupplierOffer, error) {
	return domain.SavedSupplierOffer{ID: id}, nil
}
func (s *stubProjectRepo) UpdatePartCandidateComponent(_ context.Context, _ string, _ string, _ domain.CandidateOrigin) error {
	return nil
}

type stubAssetRepo struct {
	created   *domain.ComponentAsset
	createErr error
	getResult domain.ComponentAsset
	getErr    error
	listed    []domain.ComponentAsset
	listErr   error
	updated   *domain.ComponentAsset
	updateErr error
	deleteErr error
	setErr    error
	clearErr  error
	detail    domain.ComponentWithAssets
	detailErr error

	setComponentID   string
	setAssetType     domain.AssetType
	setAssetID       string
	clearComponentID string
	clearAssetType   domain.AssetType
}

func (s *stubAssetRepo) CreateComponentAsset(_ context.Context, a domain.ComponentAsset) (domain.ComponentAsset, error) {
	if s.createErr != nil {
		return domain.ComponentAsset{}, s.createErr
	}
	s.created = &a
	return a, nil
}
func (s *stubAssetRepo) GetComponentAsset(_ context.Context, _ string) (domain.ComponentAsset, error) {
	if s.getErr != nil {
		return domain.ComponentAsset{}, s.getErr
	}
	return s.getResult, nil
}
func (s *stubAssetRepo) ListComponentAssets(_ context.Context, _ string) ([]domain.ComponentAsset, error) {
	return s.listed, s.listErr
}
func (s *stubAssetRepo) ListComponentAssetsByType(_ context.Context, _ string, _ domain.AssetType) ([]domain.ComponentAsset, error) {
	return s.listed, s.listErr
}
func (s *stubAssetRepo) UpdateComponentAsset(_ context.Context, a domain.ComponentAsset) (domain.ComponentAsset, error) {
	if s.updateErr != nil {
		return domain.ComponentAsset{}, s.updateErr
	}
	s.updated = &a
	return a, nil
}
func (s *stubAssetRepo) DeleteComponentAsset(_ context.Context, _ string) error {
	return s.deleteErr
}
func (s *stubAssetRepo) SetSelectedComponentAsset(_ context.Context, componentID string, assetType domain.AssetType, assetID string) error {
	s.setComponentID = componentID
	s.setAssetType = assetType
	s.setAssetID = assetID
	return s.setErr
}
func (s *stubAssetRepo) ClearSelectedComponentAsset(_ context.Context, componentID string, assetType domain.AssetType) error {
	s.clearComponentID = componentID
	s.clearAssetType = assetType
	return s.clearErr
}
func (s *stubAssetRepo) GetComponentWithAssets(_ context.Context, _ string) (domain.ComponentWithAssets, error) {
	return s.detail, s.detailErr
}
