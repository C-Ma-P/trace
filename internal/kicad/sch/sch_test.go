package sch_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/C-Ma-P/trace/internal/kicad"
	"github.com/C-Ma-P/trace/internal/kicad/sch"
	"github.com/C-Ma-P/trace/internal/kicad/sym"
)

const testSymLib = `(kicad_symbol_lib
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

func makeTestLib(t *testing.T) (*sym.Library, string) {
	t.Helper()
	dir := t.TempDir()
	symPath := filepath.Join(dir, "R.kicad_sym")
	if err := os.WriteFile(symPath, []byte(testSymLib), 0o644); err != nil {
		t.Fatal(err)
	}
	lib := sym.New()
	if err := lib.AddFromFile("R_10k", symPath, "R"); err != nil {
		t.Fatal(err)
	}
	return lib, symPath
}

func TestWrite_ValidSchematic(t *testing.T) {
	lib, symPath := makeTestLib(t)
	_ = symPath

	parts := []kicad.ExportedPart{
		{
			UUID:         "11111111-1111-1111-1111-111111111111",
			Reference:    "R1",
			Value:        "10k",
			SymbolLibKey: "R_10k",
			FootprintRef: "Package_SMD:R_0402",
			MPN:          "GRM0335C1H100JA01D",
			Manufacturer: "Murata",
			Package:      "0402",
			X:            50.0,
			Y:            50.0,
			InBOM:        true,
			OnBoard:      true,
		},
	}

	var buf strings.Builder
	err := sch.Write(&buf, "test_project", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", parts, lib)
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	mustContain(t, got, "kicad_sch")
	mustContain(t, got, "20230121")
	mustContain(t, got, "trace-export")
	mustContain(t, got, "A3")

	mustContain(t, got, "lib_symbols")
	mustContain(t, got, `"trace_export:R_10k"`)

	mustContain(t, got, `"trace_export:R_10k"`)
	mustContain(t, got, "(lib_id")
	mustContain(t, got, "R1")
	mustContain(t, got, "10k")
	mustContain(t, got, "Package_SMD:R_0402")
	mustContain(t, got, "GRM0335C1H100JA01D")
	mustContain(t, got, "Murata")
	mustContain(t, got, "0402")

	mustContain(t, got, "(instances")
	mustContain(t, got, "test_project")
	mustContain(t, got, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")

	mustContain(t, got, "sheet_instances")
}

func TestWrite_MultipleSymbols(t *testing.T) {
	lib, symPath := makeTestLib(t)
	_ = symPath

	parts := []kicad.ExportedPart{
		{UUID: "11111111-0000-0000-0000-000000000001", Reference: "R1", Value: "10k", SymbolLibKey: "R_10k", X: 35, Y: 40, InBOM: true, OnBoard: true},
		{UUID: "11111111-0000-0000-0000-000000000002", Reference: "R2", Value: "22k", SymbolLibKey: "R_10k", X: 65, Y: 40, InBOM: true, OnBoard: true},
	}

	var buf strings.Builder
	if err := sch.Write(&buf, "proj", "schid-1", parts, lib); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	if count := strings.Count(got, "(lib_id"); count != 2 {
		t.Errorf("expected 2 (lib_id occurrences, got %d", count)
	}
}

func TestWrite_MissingSymbolStillValid(t *testing.T) {

	lib := sym.New()
	parts := []kicad.ExportedPart{
		{UUID: "22222222-0000-0000-0000-000000000001", Reference: "U1", Value: "IC1", SymbolLibKey: "MISSING", X: 35, Y: 40, InBOM: true, OnBoard: true},
	}
	var buf strings.Builder
	err := sch.Write(&buf, "proj", "uuid-x", parts, lib)
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	mustContain(t, got, "kicad_sch")
	mustContain(t, got, "U1")
}

func TestWrite_Deterministic(t *testing.T) {
	lib, _ := makeTestLib(t)
	parts := []kicad.ExportedPart{
		{UUID: "33333333-0000-0000-0000-000000000001", Reference: "R1", Value: "10k", SymbolLibKey: "R_10k", X: 35, Y: 40, InBOM: true, OnBoard: true},
	}

	var buf1, buf2 strings.Builder
	_ = sch.Write(&buf1, "proj", "stable-uuid", parts, lib)
	_ = sch.Write(&buf2, "proj", "stable-uuid", parts, lib)

	if buf1.String() != buf2.String() {
		t.Error("two identical Write calls produced different output (not deterministic)")
	}
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\ngot (first 500 chars):\n%s", needle, truncate(haystack, 500))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
