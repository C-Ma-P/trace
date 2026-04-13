package sym_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/C-Ma-P/trace/internal/kicad/sym"
)

const twoSymLib = `(kicad_symbol_lib
  (version 20220914)
  (generator "test")
  (symbol "Alpha"
    (in_bom yes)
    (on_board yes)
    (symbol "Alpha_0_1"
      (rectangle (start -2 -1) (end 2 1)
        (stroke (width 0) (type default))
        (fill (type background))
      )
    )
    (symbol "Alpha_1_1"
      (pin input line
        (at -3.81 0 0)
        (length 1.27)
        (name "A" (effects (font (size 1.27 1.27))))
        (number "1" (effects (font (size 1.27 1.27))))
      )
    )
  )
  (symbol "Beta"
    (in_bom yes)
    (on_board yes)
    (symbol "Beta_0_1"
      (rectangle (start -1 -1) (end 1 1)
        (stroke (width 0) (type default))
        (fill (type background))
      )
    )
  )
)`

func writeTempLib(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.kicad_sym")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLibrary_AddFromFile_FirstSymbol(t *testing.T) {
	path := writeTempLib(t, twoSymLib)
	lib := sym.New()
	if err := lib.AddFromFile("alpha_key", path, ""); err != nil {
		t.Fatal(err)
	}
	keys := lib.LocalKeys()
	if len(keys) != 1 || keys[0] != "alpha_key" {
		t.Errorf("LocalKeys() = %v, want [alpha_key]", keys)
	}
}

func TestLibrary_AddFromFile_NameHint(t *testing.T) {
	path := writeTempLib(t, twoSymLib)
	lib := sym.New()
	if err := lib.AddFromFile("beta_key", path, "Beta"); err != nil {
		t.Fatal(err)
	}
	pins := lib.PinNumbers("beta_key")

	if len(pins) != 0 {
		t.Errorf("unexpected pins for beta_key: %v", pins)
	}
}

func TestLibrary_AddFromFile_PinNumbers(t *testing.T) {
	path := writeTempLib(t, twoSymLib)
	lib := sym.New()
	if err := lib.AddFromFile("alpha_key", path, "Alpha"); err != nil {
		t.Fatal(err)
	}
	pins := lib.PinNumbers("alpha_key")
	if len(pins) != 1 || pins[0] != "1" {
		t.Errorf("PinNumbers(alpha_key) = %v, want [1]", pins)
	}
}

func TestLibrary_AddFromFile_NoDuplicate(t *testing.T) {
	path := writeTempLib(t, twoSymLib)
	lib := sym.New()
	if err := lib.AddFromFile("k", path, ""); err != nil {
		t.Fatal(err)
	}

	if err := lib.AddFromFile("k", path, ""); err != nil {
		t.Fatal(err)
	}
	if len(lib.LocalKeys()) != 1 {
		t.Errorf("expected 1 key after duplicate add, got %d", len(lib.LocalKeys()))
	}
}

func TestLibrary_WriteTo_StandaloneFormat(t *testing.T) {
	path := writeTempLib(t, twoSymLib)
	lib := sym.New()
	_ = lib.AddFromFile("alpha_key", path, "Alpha")

	var buf strings.Builder
	if err := lib.WriteTo(&buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	if !strings.Contains(got, `"alpha_key"`) {
		t.Error("standalone library should contain local key name")
	}
	if strings.Contains(got, `"trace_export:`) {
		t.Error("standalone library should NOT contain library prefix")
	}

	if strings.Contains(got, `"Alpha"`) {
		t.Error("standalone library should NOT contain original symbol name Alpha")
	}
}

func TestLibrary_WriteLibSymbolsTo_SchematicFormat(t *testing.T) {
	path := writeTempLib(t, twoSymLib)
	lib := sym.New()
	_ = lib.AddFromFile("alpha_key", path, "Alpha")

	var buf strings.Builder
	if err := lib.WriteLibSymbolsTo(&buf, "trace_export"); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	if !strings.Contains(got, `"trace_export:alpha_key"`) {
		t.Error("schematic embed should contain prefixed name trace_export:alpha_key")
	}
	if strings.Contains(got, `"alpha_key"`) && !strings.Contains(got, `"trace_export:alpha_key"`) {
		t.Error("schematic embed should use prefixed name only")
	}
}

func TestLibrary_MissingFile(t *testing.T) {
	lib := sym.New()
	err := lib.AddFromFile("k", "/nonexistent/path.kicad_sym", "")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
