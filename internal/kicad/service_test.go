package kicad

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
)

type stubExporter struct {
	csv string
}

func (s stubExporter) ExportBOM(_ context.Context, _ string) ([]byte, error) {
	return []byte(s.csv), nil
}

func TestServiceListProjects_DiscoversModernProjectsOnly(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "alpha", "alpha.kicad_pro"), "")
	mustWriteFile(t, filepath.Join(root, "alpha", "alpha.kicad_sch"), "")
	mustWriteFile(t, filepath.Join(root, "legacy", "legacy.pro"), "")
	mustWriteFile(t, filepath.Join(root, "nested", "beta.kicad_pro"), "")
	mustWriteFile(t, filepath.Join(root, "nested", "beta.kicad_sch"), "")

	svc := New(stubExporter{})
	projects, err := svc.ListProjects(context.Background(), []string{root}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 modern KiCad projects, got %d", len(projects))
	}
	if projects[0].Name != "alpha" || projects[1].Name != "beta" {
		t.Fatalf("unexpected project ordering: %#v", projects)
	}
}

func TestServicePreviewImport_PreservesAllRowsIncludingWarnings(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "board.kicad_pro"), "")
	mustWriteFile(t, filepath.Join(root, "board.kicad_sch"), "")

	svc := New(stubExporter{csv: "Refs,Value,Footprint,Description,Manufacturer,MPN,ManufacturerPartNumber,PartNumber,LCSC,Datasheet,Qty,DNP\nR1,R10K,Resistor_SMD:R_0402_1005Metric,10k resistor,Yageo,RC0402,,,, ,2,\nX1,TBD,Custom:OddPart,,,,,,, ,1,\n"})
	preview, err := svc.PreviewImport(context.Background(), filepath.Join(root, "board.kicad_pro"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(preview.Rows) != 2 {
		t.Fatalf("expected 2 preview rows, got %d", len(preview.Rows))
	}
	if !preview.Rows[1].HasWarning {
		t.Fatal("expected second row to carry a warning state")
	}
	if preview.Summary.TotalRows != 2 || preview.Summary.WarningRows == 0 {
		t.Fatalf("unexpected summary: %#v", preview.Summary)
	}
}

func TestMapBOMRows_MapsResistorDraftRequirement(t *testing.T) {
	rows := mapBOMRows([]bomRow{{
		Refs:         "R1,R2",
		Value:        "10k",
		Footprint:    "Resistor_SMD:R_0402_1005Metric",
		Description:  "10k resistor",
		Manufacturer: "Yageo",
		MPN:          "RC0402FR-0710KL",
		Quantity:     2,
	}})
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row.Requirement.Category != domain.CategoryResistor {
		t.Fatalf("expected resistor category, got %q", row.Requirement.Category)
	}
	if row.Requirement.Quantity != 2 {
		t.Fatalf("expected requirement quantity 2, got %d", row.Requirement.Quantity)
	}
	if got := constraintByKey(row.Requirement.Constraints, registry.AttrResistanceOhms); got == nil || got.Number == nil || *got.Number != 10000 {
		t.Fatalf("expected resistance constraint to be inferred, got %#v", got)
	}
	if got := constraintByKey(row.Requirement.Constraints, registry.AttrPackage); got == nil || got.Text == nil || *got.Text == "" {
		t.Fatalf("expected package constraint to be inferred, got %#v", got)
	}
	if row.HasWarning {
		t.Fatalf("did not expect a warning for a clean resistor row: %#v", row.WarningMessages)
	}
}

func constraintByKey(constraints []domain.RequirementConstraint, key string) *domain.RequirementConstraint {
	for i := range constraints {
		if constraints[i].Key == key {
			return &constraints[i]
		}
	}
	return nil
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
