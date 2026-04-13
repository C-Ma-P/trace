package project_test

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/C-Ma-P/trace/internal/kicad"
	"github.com/C-Ma-P/trace/internal/kicad/project"
)

const resistorSymLib = `(kicad_symbol_lib
  (version 20220914)
  (generator "test")
  (symbol "R"
    (in_bom yes)
    (on_board yes)
    (symbol "R_0_1"
      (rectangle (start -1 -2) (end 1 2)
        (stroke (width 0.2) (type default))
        (fill (type none))
      )
    )
    (symbol "R_1_1"
      (pin passive line
        (at 0 3.81 270)
        (length 1.778)
        (name "~" (effects (font (size 1.27 1.27))))
        (number "1" (effects (font (size 1.27 1.27))))
      )
      (pin passive line
        (at 0 -3.81 90)
        (length 1.27)
        (name "~" (effects (font (size 1.27 1.27))))
        (number "2" (effects (font (size 1.27 1.27))))
      )
    )
  )
)`

const capacitorSymLib = `(kicad_symbol_lib
  (version 20220914)
  (generator "test")
  (symbol "C"
    (in_bom yes)
    (on_board yes)
    (symbol "C_0_1"
      (rectangle (start -1 -1) (end 1 1)
        (stroke (width 0.2) (type default))
        (fill (type none))
      )
    )
  )
)`

func writeTempSym(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name+".kicad_sym")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func buildInput(t *testing.T) kicad.ExportInput {
	t.Helper()
	rPath := writeTempSym(t, "R_10k", resistorSymLib)
	cPath := writeTempSym(t, "C_100n", capacitorSymLib)

	return kicad.ExportInput{
		ProjectName:   "Test Board",
		SchematicUUID: "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		Parts: []kicad.ExportedPart{
			{
				UUID:          "11111111-0000-0000-0000-000000000001",
				Reference:     "R1",
				Value:         "10k",
				SymbolLibKey:  "R_10k",
				SymbolSrcPath: rPath,
				FootprintRef:  "Package_SMD:R_0402",
				MPN:           "GRM0335C1H100JA01D",
				Manufacturer:  "Murata",
				Package:       "0402",
				InBOM:         true,
				OnBoard:       true,
			},
			{
				UUID:          "22222222-0000-0000-0000-000000000002",
				Reference:     "C1",
				Value:         "100nF",
				SymbolLibKey:  "C_100n",
				SymbolSrcPath: cPath,
				FootprintRef:  "Package_SMD:C_0402",
				MPN:           "GRM155R61A104KA01D",
				Manufacturer:  "Murata",
				Package:       "0402",
				InBOM:         true,
				OnBoard:       true,
			},
		},
	}
}

func TestExport_CreatesZip(t *testing.T) {
	input := buildInput(t)
	output, err := project.Export(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(output.Dir)
		os.Remove(output.ZipPath)
	})

	if _, err := os.Stat(output.ZipPath); err != nil {
		t.Errorf("zip file not found: %v", err)
	}
}

func TestExport_ZipContainsRequiredFiles(t *testing.T) {
	input := buildInput(t)
	output, err := project.Export(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(output.Dir)
		os.Remove(output.ZipPath)
	})

	r, err := zip.OpenReader(output.ZipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()

	fileNames := make(map[string]bool)
	for _, f := range r.File {
		fileNames[filepath.Base(f.Name)] = true
		t.Logf("zip entry: %s", f.Name)
	}

	required := []string{
		"Test_Board.kicad_pro",
		"Test_Board.kicad_sch",
		"trace_export.kicad_sym",
		"sym-lib-table",
	}
	for _, name := range required {
		if !fileNames[name] {
			t.Errorf("zip is missing required file: %s", name)
		}
	}
}

func TestExport_ProFileIsValidJSON(t *testing.T) {
	input := buildInput(t)
	output, err := project.Export(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(output.Dir)
		os.Remove(output.ZipPath)
	})

	proPath := filepath.Join(output.Dir, "Test_Board.kicad_pro")
	data, err := os.ReadFile(proPath)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Errorf(".kicad_pro is not valid JSON: %v\n%s", err, data)
	}

	for _, key := range []string{"meta", "schematic", "libraries"} {
		if _, ok := obj[key]; !ok {
			t.Errorf(".kicad_pro is missing key %q", key)
		}
	}
}

func TestExport_SymLibTableContainsTraceExport(t *testing.T) {
	input := buildInput(t)
	output, err := project.Export(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(output.Dir)
		os.Remove(output.ZipPath)
	})

	data, err := os.ReadFile(filepath.Join(output.Dir, "sym-lib-table"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "trace_export") {
		t.Error("sym-lib-table does not reference trace_export")
	}
	if !strings.Contains(string(data), "${KIPRJMOD}") {
		t.Error("sym-lib-table should use ${KIPRJMOD} variable")
	}
}

func TestExport_SchematicContainsBothParts(t *testing.T) {
	input := buildInput(t)
	output, err := project.Export(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(output.Dir)
		os.Remove(output.ZipPath)
	})

	data, err := os.ReadFile(filepath.Join(output.Dir, "Test_Board.kicad_sch"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	for _, expect := range []string{"R1", "C1", "10k", "100nF", "GRM0335C1H100JA01D", "GRM155R61A104KA01D"} {
		if !strings.Contains(content, expect) {
			t.Errorf("schematic does not contain expected value %q", expect)
		}
	}
}

func TestExport_GridPlacement(t *testing.T) {

	rPath := writeTempSym(t, "R", resistorSymLib)

	parts := make([]kicad.ExportedPart, 10)
	for i := range parts {
		parts[i] = kicad.ExportedPart{
			UUID:          fmt.Sprintf("aaaaaaaa-0000-0000-0000-%012d", i),
			Reference:     fmt.Sprintf("R%d", i+1),
			Value:         "10k",
			SymbolLibKey:  "R_key",
			SymbolSrcPath: rPath,
			InBOM:         true,
			OnBoard:       true,
		}
	}

	input := kicad.ExportInput{
		ProjectName:   "grid_test",
		SchematicUUID: "stable-uuid-1234",
		Parts:         parts,
	}
	output, err := project.Export(input)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.RemoveAll(output.Dir)
		os.Remove(output.ZipPath)
	})

	data, err := os.ReadFile(filepath.Join(output.Dir, "grid_test.kicad_sch"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "(at 35 40 0)") {
		t.Errorf("first part not placed at expected grid origin 35,40\n(first 800 chars):\n%s", content[:min(800, len(content))])
	}
}

func TestExport_Deterministic(t *testing.T) {
	input := buildInput(t)

	input.SchematicUUID = "stable-deterministic-uuid"

	out1, err := project.Export(input)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(out1.Dir)
	defer os.Remove(out1.ZipPath)

	out2, err := project.Export(input)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(out2.Dir)
	defer os.Remove(out2.ZipPath)

	data1, _ := os.ReadFile(filepath.Join(out1.Dir, "Test_Board.kicad_sch"))
	data2, _ := os.ReadFile(filepath.Join(out2.Dir, "Test_Board.kicad_sch"))

	if string(data1) != string(data2) {
		t.Error("two identical Export calls produced different schematic content")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
