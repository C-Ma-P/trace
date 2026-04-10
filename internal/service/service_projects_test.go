package service_test

import (
	"context"
	"errors"
	"testing"

	"componentmanager/internal/domain"
	"componentmanager/internal/domain/registry"
	"componentmanager/internal/kicad"
	"componentmanager/internal/service"
)

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

// --- ErrNotFound propagation ---

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
