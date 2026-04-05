package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"componentmanager/internal/domain"
	"componentmanager/internal/domain/registry"
	"componentmanager/internal/kicad"
	"componentmanager/internal/service"
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

func TestCreateProject_ValidConstraints_Persisted(t *testing.T) {
	repo := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, repo, &stubAssetRepo{})

	n := 10000.0
	project := domain.Project{
		Name: "test",
		Requirements: []domain.ProjectRequirement{
			{
				Name:     "R1",
				Category: domain.CategoryResistor,
				Quantity: 1,
				Constraints: []domain.RequirementConstraint{
					{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Operator: domain.OperatorEqual, Number: &n, Unit: "ohm"},
				},
			},
		},
	}

	result, err := svc.CreateProject(context.Background(), project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID == "" {
		t.Error("expected project ID to be assigned")
	}
	if repo.created == nil {
		t.Fatal("expected project to be persisted")
	}
}

func TestCreateProject_InvalidConstraintKey_Rejected(t *testing.T) {
	repo := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, repo, &stubAssetRepo{})

	n := 1.0
	project := domain.Project{
		Name: "test",
		Requirements: []domain.ProjectRequirement{
			{
				Name:     "C1",
				Category: domain.CategoryCapacitor,
				Quantity: 1,
				Constraints: []domain.RequirementConstraint{
					{Key: "unknown_key", ValueType: domain.ValueTypeNumber, Operator: domain.OperatorEqual, Number: &n},
				},
			},
		},
	}

	_, err := svc.CreateProject(context.Background(), project)
	if err == nil {
		t.Fatal("expected error for unknown constraint key")
	}
	var target domain.ErrUnknownConstraint
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownConstraint, got %T: %v", err, err)
	}
	if repo.created != nil {
		t.Error("project should not have been persisted")
	}
}

func TestCreateProject_InvalidOperator_Rejected(t *testing.T) {
	repo := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, repo, &stubAssetRepo{})

	v := "0402"
	project := domain.Project{
		Name: "test",
		Requirements: []domain.ProjectRequirement{
			{
				Name:     "R1",
				Category: domain.CategoryResistor,
				Quantity: 1,
				Constraints: []domain.RequirementConstraint{
					{Key: registry.AttrPackage, ValueType: domain.ValueTypeText, Operator: domain.OperatorGTE, Text: &v},
				},
			},
		},
	}

	_, err := svc.CreateProject(context.Background(), project)
	if err == nil {
		t.Fatal("expected error for invalid operator on text constraint")
	}
	var target domain.ErrInvalidOperator
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrInvalidOperator, got %T: %v", err, err)
	}
	if repo.created != nil {
		t.Error("project should not have been persisted")
	}
}

func TestCreateProject_NoRequirements_OK(t *testing.T) {
	repo := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, repo, &stubAssetRepo{})

	project := domain.Project{Name: "empty"}
	_, err := svc.CreateProject(context.Background(), project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSyncCanonicalAttributeDefinitions_AllCategoriesUpserted(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	if err := svc.SyncCanonicalAttributeDefinitions(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(comp.upserted) == 0 {
		t.Fatal("expected definitions to be upserted")
	}

	byCategory := make(map[domain.Category]int)
	for _, def := range comp.upserted {
		byCategory[def.Category]++
	}

	for _, cat := range []domain.Category{domain.CategoryResistor, domain.CategoryCapacitor, domain.CategoryInductor} {
		if byCategory[cat] == 0 {
			t.Errorf("no definitions upserted for category %q", cat)
		}
	}
}

func TestSyncCanonicalAttributeDefinitions_InductorIncluded(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	if err := svc.SyncCanonicalAttributeDefinitions(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var inductorKeys []string
	for _, def := range comp.upserted {
		if def.Category == domain.CategoryInductor {
			inductorKeys = append(inductorKeys, def.Key)
		}
	}

	if len(inductorKeys) == 0 {
		t.Fatal("no inductor definitions were synced")
	}

	wantKeys := map[string]bool{
		"inductance_h": false, "current_a": false, "dcr_ohms": false,
		"tolerance_percent": false, "package": false, "inductor_type": false,
	}
	for _, k := range inductorKeys {
		wantKeys[k] = true
	}
	for key, found := range wantKeys {
		if !found {
			t.Errorf("inductor key %q was not synced", key)
		}
	}
}

// --- UpdateComponentMetadata ---

func TestUpdateComponentMetadata_Persisted(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	c := domain.Component{
		ID:           "cid-1",
		Category:     domain.CategoryResistor,
		MPN:          "RC0402",
		Manufacturer: "Yageo",
		Package:      "0402",
		Description:  "basic resistor",
	}

	result, err := svc.UpdateComponentMetadata(context.Background(), c)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.updatedComp == nil {
		t.Fatal("expected UpdateComponentMetadata to be called on repository")
	}
	if result.ID != c.ID {
		t.Errorf("expected ID %q, got %q", c.ID, result.ID)
	}
	if comp.updatedComp.Category != domain.CategoryResistor {
		t.Error("expected category to be unchanged")
	}
}

// --- ReplaceComponentAttributes ---

func TestReplaceComponentAttributes_InvalidAttr_Rejected(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{
			ID:       "cid-1",
			Category: domain.CategoryResistor,
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	n := 47.0
	attrs := []domain.AttributeValue{
		// wrong unit for resistance (should be "ohm" but passing "milli-ohm")
		{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Number: &n, Unit: "milli-ohm"},
	}

	err := svc.ReplaceComponentAttributes(context.Background(), "cid-1", attrs)
	if err == nil {
		t.Fatal("expected error for unit mismatch")
	}
	var target domain.ErrAttributeUnitMismatch
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrAttributeUnitMismatch, got %T: %v", err, err)
	}
	if comp.replacedID != "" {
		t.Error("repository ReplaceComponentAttributes should not have been called")
	}
}

func TestReplaceComponentAttributes_Valid(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{
			ID:       "cid-1",
			Category: domain.CategoryResistor,
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	n := 10000.0
	attrs := []domain.AttributeValue{
		{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Number: &n, Unit: "ohm"},
	}

	err := svc.ReplaceComponentAttributes(context.Background(), "cid-1", attrs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.replacedID != "cid-1" {
		t.Errorf("expected replacedID %q, got %q", "cid-1", comp.replacedID)
	}
	if len(comp.replacedAttrs) != 1 {
		t.Errorf("expected 1 attr passed to repo, got %d", len(comp.replacedAttrs))
	}
}

func TestReplaceComponentAttributes_UnknownKey_Rejected(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{
			ID:       "cid-1",
			Category: domain.CategoryCapacitor,
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	n := 1.0
	attrs := []domain.AttributeValue{
		{Key: "not_a_real_key", ValueType: domain.ValueTypeNumber, Number: &n},
	}

	err := svc.ReplaceComponentAttributes(context.Background(), "cid-1", attrs)
	if err == nil {
		t.Fatal("expected error for unknown attribute key")
	}
	var target domain.ErrUnknownAttribute
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownAttribute, got %T: %v", err, err)
	}
}

// --- FindComponents ---

func TestFindComponents_EmptyFilter_DelegatesToRepo(t *testing.T) {
	comp := &stubComponentRepo{
		findResult: []domain.Component{
			{ID: "c1", Category: domain.CategoryResistor, MPN: "R1"},
			{ID: "c2", Category: domain.CategoryCapacitor, MPN: "C1"},
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	results, err := svc.FindComponents(context.Background(), domain.ComponentFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}
}

func TestFindComponents_ByCategory(t *testing.T) {
	cat := domain.CategoryResistor
	comp := &stubComponentRepo{
		findResult: []domain.Component{
			{ID: "c1", Category: domain.CategoryResistor, MPN: "R1"},
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{Category: &cat}
	_, err := svc.FindComponents(context.Background(), filter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if comp.lastFilter.Category == nil || *comp.lastFilter.Category != domain.CategoryResistor {
		t.Error("expected category filter to be passed to repository")
	}
}

func TestFindComponents_ByManufacturer(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{Manufacturer: "Yageo"}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.Manufacturer != "Yageo" {
		t.Errorf("expected Manufacturer %q passed to repo, got %q", "Yageo", comp.lastFilter.Manufacturer)
	}
}

func TestFindComponents_ByMPN(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{MPN: "RC04"}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.MPN != "RC04" {
		t.Errorf("expected MPN %q passed to repo, got %q", "RC04", comp.lastFilter.MPN)
	}
}

func TestFindComponents_ByPackage(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{Package: "0402"}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.Package != "0402" {
		t.Errorf("expected Package %q passed to repo, got %q", "0402", comp.lastFilter.Package)
	}
}

func TestFindComponents_ByText(t *testing.T) {
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{Text: "ceramic"}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.Text != "ceramic" {
		t.Errorf("expected Text %q passed to repo, got %q", "ceramic", comp.lastFilter.Text)
	}
}

func TestFindComponents_CombinedFilter(t *testing.T) {
	cat := domain.CategoryCapacitor
	comp := &stubComponentRepo{}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	filter := domain.ComponentFilter{
		Category:     &cat,
		Manufacturer: "Murata",
		Package:      "0402",
	}
	svc.FindComponents(context.Background(), filter)
	if comp.lastFilter.Manufacturer != "Murata" || comp.lastFilter.Package != "0402" {
		t.Error("combined filter fields not passed to repository")
	}
}

func TestFindComponents_ReturnsAttributesFromRepo(t *testing.T) {
	n := 100.0
	comp := &stubComponentRepo{
		findResult: []domain.Component{
			{
				ID:       "c1",
				Category: domain.CategoryResistor,
				Attributes: []domain.AttributeValue{
					{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Number: &n, Unit: "ohm"},
				},
			},
		},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	results, err := svc.FindComponents(context.Background(), domain.ComponentFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}
	if len(results[0].Attributes) == 0 {
		t.Error("expected attributes to be present on returned components")
	}
}

// --- ListProjects ---

func TestListProjects_DelegatesToRepo(t *testing.T) {
	proj := &stubProjectRepo{
		listed: []domain.Project{
			{ID: "p1", Name: "Alpha"},
			{ID: "p2", Name: "Beta"},
		},
	}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	results, err := svc.ListProjects(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 projects, got %d", len(results))
	}
}

// --- UpdateProject ---

func TestUpdateProject_Persisted(t *testing.T) {
	proj := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	p := domain.Project{ID: "p1", Name: "Updated Name", Description: "new desc"}
	result, err := svc.UpdateProject(context.Background(), p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.updatedProj == nil {
		t.Fatal("expected UpdateProject to be called on repository")
	}
	if result.Name != "Updated Name" {
		t.Errorf("expected name %q, got %q", "Updated Name", result.Name)
	}
}

// --- ReplaceProjectRequirements ---

func TestReplaceProjectRequirements_AssignsIDs(t *testing.T) {
	proj := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	reqs := []domain.ProjectRequirement{
		{Name: "R1", Category: domain.CategoryResistor, Quantity: 1},
	}

	err := svc.ReplaceProjectRequirements(context.Background(), "p1", reqs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.replacedForProject != "p1" {
		t.Errorf("expected projectID %q, got %q", "p1", proj.replacedForProject)
	}
	if len(proj.replacedReqs) != 1 {
		t.Fatalf("expected 1 requirement, got %d", len(proj.replacedReqs))
	}
	if proj.replacedReqs[0].ID == "" {
		t.Error("expected requirement ID to be assigned")
	}
	if proj.replacedReqs[0].ProjectID != "p1" {
		t.Errorf("expected ProjectID %q, got %q", "p1", proj.replacedReqs[0].ProjectID)
	}
}

func TestReplaceProjectRequirements_InvalidConstraint_Rejected(t *testing.T) {
	proj := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	n := 1.0
	reqs := []domain.ProjectRequirement{
		{
			Name:     "C1",
			Category: domain.CategoryCapacitor,
			Quantity: 1,
			Constraints: []domain.RequirementConstraint{
				{Key: "not_a_capacitor_key", ValueType: domain.ValueTypeNumber, Operator: domain.OperatorEqual, Number: &n},
			},
		},
	}

	err := svc.ReplaceProjectRequirements(context.Background(), "p1", reqs)
	if err == nil {
		t.Fatal("expected error for unknown constraint key")
	}
	var target domain.ErrUnknownConstraint
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownConstraint, got %T: %v", err, err)
	}
	if proj.replacedForProject != "" {
		t.Error("repo ReplaceProjectRequirements should not have been called")
	}
}

func TestReplaceProjectRequirements_ValidConstraints_Persisted(t *testing.T) {
	proj := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	n := 100.0
	reqs := []domain.ProjectRequirement{
		{
			Name:     "R1",
			Category: domain.CategoryResistor,
			Quantity: 2,
			Constraints: []domain.RequirementConstraint{
				{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Operator: domain.OperatorEqual, Number: &n, Unit: "ohm"},
			},
		},
	}

	err := svc.ReplaceProjectRequirements(context.Background(), "p1", reqs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.replacedForProject != "p1" {
		t.Errorf("expected projectID %q, got %q", "p1", proj.replacedForProject)
	}
}

func TestAddProjectRequirements_AssignsIDs(t *testing.T) {
	proj := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	reqs := []domain.ProjectRequirement{{Name: "R1", Category: domain.CategoryResistor, Quantity: 1}}

	err := svc.AddProjectRequirements(context.Background(), "p1", reqs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proj.addedForProject != "p1" {
		t.Errorf("expected projectID %q, got %q", "p1", proj.addedForProject)
	}
	if len(proj.addedReqs) != 1 || proj.addedReqs[0].ID == "" {
		t.Fatalf("expected appended requirement with assigned ID")
	}
	if proj.addedReqs[0].ProjectID != "p1" {
		t.Errorf("expected project ID to be assigned, got %q", proj.addedReqs[0].ProjectID)
	}
}

func TestImportKiCadProject_ExistingProject_AppendsAndUpdatesProvenance(t *testing.T) {
	proj := &stubProjectRepo{
		getResult: domain.Project{ID: "p1", Name: "Existing"},
	}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	result, err := svc.ImportKiCadProject(context.Background(), kicad.ImportCommitInput{
		TargetMode:        kicad.ImportTargetModeExisting,
		ExistingProjectID: "p1",
		SourceProjectPath: "/tmp/demo.kicad_pro",
		Rows: []kicad.ImportPreviewRow{{
			RowID:      "row-001",
			Included:   true,
			SourceRefs: "R1,R2",
			Requirement: kicad.DraftRequirement{
				Name:     "10k resistor",
				Category: domain.CategoryResistor,
				Quantity: 2,
			},
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "p1" {
		t.Fatalf("expected existing project response, got %q", result.ID)
	}
	if proj.addedForProject != "p1" || len(proj.addedReqs) != 1 {
		t.Fatalf("expected requirements to be appended")
	}
	if proj.importMetaProject != "p1" {
		t.Fatalf("expected import metadata to be updated for project p1")
	}
	if proj.importSourceType == nil || *proj.importSourceType != "kicad" {
		t.Fatalf("expected import source type to be set to kicad")
	}
	if proj.importSourcePath == nil || *proj.importSourcePath != "/tmp/demo.kicad_pro" {
		t.Fatalf("expected import source path to be recorded")
	}
	if proj.importedAt == nil {
		t.Fatalf("expected imported_at to be recorded")
	}
}

func TestImportKiCadProject_NewProject_CreatesProjectWithProvenance(t *testing.T) {
	proj := &stubProjectRepo{}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	result, err := svc.ImportKiCadProject(context.Background(), kicad.ImportCommitInput{
		TargetMode:            kicad.ImportTargetModeNew,
		NewProjectID:          "proj-kicad",
		NewProjectName:        "Imported Board",
		NewProjectDescription: "From KiCad",
		SourceProjectPath:     "/tmp/demo.kicad_pro",
		Rows: []kicad.ImportPreviewRow{{
			RowID:      "row-001",
			Included:   true,
			SourceRefs: "C1",
			Requirement: kicad.DraftRequirement{
				Name:     "100n capacitor",
				Category: domain.CategoryCapacitor,
				Quantity: 1,
			},
		}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "proj-kicad" {
		t.Fatalf("expected fixed project ID to be used, got %q", result.ID)
	}
	if proj.created == nil {
		t.Fatal("expected project to be created")
	}
	if proj.created.ImportSourceType == nil || *proj.created.ImportSourceType != "kicad" {
		t.Fatal("expected project provenance to be set")
	}
	if proj.created.ImportedAt == nil {
		t.Fatal("expected imported_at to be set on the new project")
	}
	if len(proj.created.Requirements) != 1 {
		t.Fatal("expected imported requirements to be attached to the new project")
	}
}

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

// --- ErrNotFound propagation ---

func TestGetComponent_ErrNotFound(t *testing.T) {
	comp := &stubComponentRepo{getErr: domain.ErrNotFound{ID: "cid-x"}}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.GetComponent(context.Background(), "cid-x")
	var target domain.ErrNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrNotFound, got %T: %v", err, err)
	}
	if target.ID != "cid-x" {
		t.Errorf("expected ID %q, got %q", "cid-x", target.ID)
	}
}

func TestUpdateComponentMetadata_ErrNotFound(t *testing.T) {
	comp := &stubComponentRepo{updateCompErr: domain.ErrNotFound{ID: "cid-x"}}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.UpdateComponentMetadata(context.Background(), domain.Component{ID: "cid-x", MPN: "X"})
	var target domain.ErrNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrNotFound, got %T: %v", err, err)
	}
	if target.ID != "cid-x" {
		t.Errorf("expected ID %q, got %q", "cid-x", target.ID)
	}
}

func TestGetProject_ErrNotFound(t *testing.T) {
	proj := &stubProjectRepo{getProjectErr: domain.ErrNotFound{ID: "proj-x"}}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	_, err := svc.GetProject(context.Background(), "proj-x")
	var target domain.ErrNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrNotFound, got %T: %v", err, err)
	}
	if target.ID != "proj-x" {
		t.Errorf("expected ID %q, got %q", "proj-x", target.ID)
	}
}

func TestUpdateProject_ErrNotFound(t *testing.T) {
	proj := &stubProjectRepo{updateProjectErr: domain.ErrNotFound{ID: "proj-x"}}
	svc := service.New(&stubComponentRepo{}, proj, &stubAssetRepo{})

	_, err := svc.UpdateProject(context.Background(), domain.Project{ID: "proj-x", Name: "X"})
	var target domain.ErrNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrNotFound, got %T: %v", err, err)
	}
	if target.ID != "proj-x" {
		t.Errorf("expected ID %q, got %q", "proj-x", target.ID)
	}
}

// --- Component Assets ---

func TestCreateComponentAsset_Valid(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{ID: "cid-1", Category: domain.CategoryResistor},
	}
	assets := &stubAssetRepo{}
	svc := service.New(comp, &stubProjectRepo{}, assets)

	a, err := svc.CreateComponentAsset(context.Background(), domain.ComponentAsset{
		ComponentID: "cid-1",
		AssetType:   domain.AssetTypeSymbol,
		Label:       "R_0402",
		URLOrPath:   "/symbols/R_0402.kicad_sym",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID == "" {
		t.Error("expected ID to be assigned")
	}
	if a.Source != "manual" {
		t.Errorf("expected default source %q, got %q", "manual", a.Source)
	}
	if a.Status != domain.AssetStatusCandidate {
		t.Errorf("expected default status %q, got %q", domain.AssetStatusCandidate, a.Status)
	}
	if assets.created == nil {
		t.Error("expected asset to be persisted via repo")
	}
}

func TestCreateComponentAsset_InvalidType_Rejected(t *testing.T) {
	comp := &stubComponentRepo{
		getResult: domain.Component{ID: "cid-1"},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.CreateComponentAsset(context.Background(), domain.ComponentAsset{
		ComponentID: "cid-1",
		AssetType:   "invalid_type",
	})
	if err == nil {
		t.Fatal("expected error for invalid asset type")
	}
}

func TestCreateComponentAsset_ComponentNotFound_Rejected(t *testing.T) {
	comp := &stubComponentRepo{
		getErr: domain.ErrNotFound{ID: "cid-missing"},
	}
	svc := service.New(comp, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.CreateComponentAsset(context.Background(), domain.ComponentAsset{
		ComponentID: "cid-missing",
		AssetType:   domain.AssetTypeFootprint,
	})
	if err == nil {
		t.Fatal("expected error when component not found")
	}
	var target domain.ErrNotFound
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrNotFound, got %T: %v", err, err)
	}
}

func TestSetSelectedComponentAsset_InvalidType_Rejected(t *testing.T) {
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, &stubAssetRepo{})

	err := svc.SetSelectedComponentAsset(context.Background(), "cid-1", "bad_type", "asset-1")
	if err == nil {
		t.Fatal("expected error for invalid asset type")
	}
}

func TestSetSelectedComponentAsset_Valid_Delegated(t *testing.T) {
	assets := &stubAssetRepo{}
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, assets)

	err := svc.SetSelectedComponentAsset(context.Background(), "cid-1", domain.AssetTypeSymbol, "asset-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assets.setComponentID != "cid-1" {
		t.Errorf("expected componentID %q, got %q", "cid-1", assets.setComponentID)
	}
	if assets.setAssetType != domain.AssetTypeSymbol {
		t.Errorf("expected assetType %q, got %q", domain.AssetTypeSymbol, assets.setAssetType)
	}
	if assets.setAssetID != "asset-1" {
		t.Errorf("expected assetID %q, got %q", "asset-1", assets.setAssetID)
	}
}

func TestClearSelectedComponentAsset_Valid_Delegated(t *testing.T) {
	assets := &stubAssetRepo{}
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, assets)

	err := svc.ClearSelectedComponentAsset(context.Background(), "cid-1", domain.AssetTypeFootprint)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if assets.clearComponentID != "cid-1" {
		t.Errorf("expected componentID %q, got %q", "cid-1", assets.clearComponentID)
	}
	if assets.clearAssetType != domain.AssetTypeFootprint {
		t.Errorf("expected assetType %q, got %q", domain.AssetTypeFootprint, assets.clearAssetType)
	}
}

func TestClearSelectedComponentAsset_InvalidType_Rejected(t *testing.T) {
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, &stubAssetRepo{})

	err := svc.ClearSelectedComponentAsset(context.Background(), "cid-1", "nope")
	if err == nil {
		t.Fatal("expected error for invalid asset type")
	}
}

func TestUpdateComponentAssetStatus_Valid(t *testing.T) {
	assets := &stubAssetRepo{
		getResult: domain.ComponentAsset{
			ID:        "asset-1",
			AssetType: domain.AssetTypeDatasheet,
			Status:    domain.AssetStatusCandidate,
		},
	}
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, assets)

	result, err := svc.UpdateComponentAssetStatus(context.Background(), "asset-1", domain.AssetStatusVerified)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != domain.AssetStatusVerified {
		t.Errorf("expected status %q, got %q", domain.AssetStatusVerified, result.Status)
	}
}

func TestUpdateComponentAssetStatus_InvalidStatus_Rejected(t *testing.T) {
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, &stubAssetRepo{})

	_, err := svc.UpdateComponentAssetStatus(context.Background(), "asset-1", "bogus")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestGetComponentWithAssets_Delegated(t *testing.T) {
	assets := &stubAssetRepo{
		detail: domain.ComponentWithAssets{
			Component: domain.Component{ID: "cid-1"},
			Assets:    []domain.ComponentAsset{{ID: "a1", AssetType: domain.AssetTypeSymbol}},
		},
	}
	svc := service.New(&stubComponentRepo{}, &stubProjectRepo{}, assets)

	detail, err := svc.GetComponentWithAssets(context.Background(), "cid-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Component.ID != "cid-1" {
		t.Errorf("expected component ID %q, got %q", "cid-1", detail.Component.ID)
	}
	if len(detail.Assets) != 1 {
		t.Errorf("expected 1 asset, got %d", len(detail.Assets))
	}
}
