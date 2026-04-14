package sexpr_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/C-Ma-P/trace/internal/kicad/sexpr"
)

func TestWriter_SimpleLines(t *testing.T) {
	w := sexpr.New()
	w.Enter("kicad_sch")
	w.Line("(version 20230121)")
	w.Line(fmt.Sprintf("(generator %s)", sexpr.Q("trace-export")))
	w.Leave()

	got := w.String()
	mustContain(t, got, "(kicad_sch")
	mustContain(t, got, "(version 20230121)")
	mustContain(t, got, `(generator "trace-export")`)
}

func TestWriter_Indentation(t *testing.T) {
	w := sexpr.New()
	w.Enter("outer")
	w.Enter("inner")
	w.Line("(leaf yes)")
	w.Leave()
	w.Leave()

	lines := strings.Split(strings.TrimRight(w.String(), "\n"), "\n")

	if strings.HasPrefix(lines[0], " ") {
		t.Errorf("first line should have no indent, got %q", lines[0])
	}

	if !strings.HasPrefix(lines[1], "  ") {
		t.Errorf("inner should be indented by 2 spaces, got %q", lines[1])
	}

	if !strings.HasPrefix(lines[2], "    ") {
		t.Errorf("leaf should be indented by 4 spaces, got %q", lines[2])
	}
}

func TestWriter_RawBlock(t *testing.T) {
	w := sexpr.New()
	w.Enter("outer")
	w.Raw("(symbol \"R\"\n  (in_bom yes)\n)")
	w.Leave()

	got := w.String()
	mustContain(t, got, "(symbol")
	mustContain(t, got, "(in_bom yes)")
}

func TestQ_Escaping(t *testing.T) {
	cases := []struct{ in, want string }{
		{`hello`, `"hello"`},
		{`say "hi"`, `"say \"hi\""`},
		{`back\slash`, `"back\\slash"`},
	}
	for _, c := range cases {
		got := sexpr.Q(c.in)
		if got != c.want {
			t.Errorf("Q(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFmtNum(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0"},
		{1.0, "1"},
		{1.5, "1.5"},
		{-2.54, "-2.54"},
		{25.4001, "25.4001"},
		{0.000001, "0.000001"},
	}
	for _, c := range cases {
		got := sexpr.FmtNum(c.in)
		if got != c.want {
			t.Errorf("FmtNum(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

const minimalSymLib = `(kicad_symbol_lib
  (version 20220914)
  (generator "test")
  (symbol "TestPart"
    (in_bom yes)
    (on_board yes)
    (symbol "TestPart_0_1"
      (rectangle (start -1 -1) (end 1 1)
        (stroke (width 0) (type default))
        (fill (type background))
      )
    )
    (symbol "TestPart_1_1"
      (pin passive line
        (at 0 2.54 270)
        (length 1.27)
        (name "~" (effects (font (size 1.27 1.27))))
        (number "1" (effects (font (size 1.27 1.27))))
      )
      (pin passive line
        (at 0 -2.54 90)
        (length 1.27)
        (name "~" (effects (font (size 1.27 1.27))))
        (number "2" (effects (font (size 1.27 1.27))))
      )
    )
  )
)`

func TestExtractSymbolBlocks_Single(t *testing.T) {
	blocks, err := sexpr.ExtractSymbolBlocks([]byte(minimalSymLib))
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(blocks))
	}
	name := sexpr.BlockName(blocks[0])
	if name != "TestPart" {
		t.Errorf("BlockName = %q, want %q", name, "TestPart")
	}
}

func TestExtractSymbolBlocks_MultipleSymbols(t *testing.T) {
	src := `(kicad_symbol_lib
  (version 20220914)
  (symbol "Alpha" (in_bom yes) (on_board yes))
  (symbol "Beta"  (in_bom yes) (on_board yes))
)`
	blocks, err := sexpr.ExtractSymbolBlocks([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(blocks))
	}
	if sexpr.BlockName(blocks[0]) != "Alpha" {
		t.Errorf("first block name = %q, want Alpha", sexpr.BlockName(blocks[0]))
	}
	if sexpr.BlockName(blocks[1]) != "Beta" {
		t.Errorf("second block name = %q, want Beta", sexpr.BlockName(blocks[1]))
	}
}

func TestExtractSymbolBlocks_Error_NotALib(t *testing.T) {
	_, err := sexpr.ExtractSymbolBlocks([]byte(`(kicad_sch (version 20230121))`))
	if err == nil {
		t.Error("expected error for non-library input")
	}
}

func TestRenameSymbolBlock(t *testing.T) {
	blocks, err := sexpr.ExtractSymbolBlocks([]byte(minimalSymLib))
	if err != nil {
		t.Fatal(err)
	}
	renamed := sexpr.RenameSymbolBlock(blocks[0], "TestPart", "trace_export:TestPart")
	serialised := sexpr.SerializeTokens(renamed)

	if !strings.Contains(serialised, `"trace_export:TestPart"`) {
		t.Error("renamed block should contain trace_export:TestPart")
	}
	if !strings.Contains(serialised, `"TestPart_0_1"`) {
		t.Error("renamed block should contain sub-symbol TestPart_0_1 (unprefixed)")
	}
	if !strings.Contains(serialised, `"TestPart_1_1"`) {
		t.Error("renamed block should contain sub-symbol TestPart_1_1 (unprefixed)")
	}
	if strings.Contains(serialised, `"trace_export:TestPart_0_1"`) {
		t.Error("sub-symbol should NOT have library prefix")
	}

	if strings.Contains(serialised, `"TestPart"`) {
		t.Error("renamed block should NOT contain original name TestPart as a standalone string")
	}
}

func TestPinNumbers(t *testing.T) {
	blocks, err := sexpr.ExtractSymbolBlocks([]byte(minimalSymLib))
	if err != nil {
		t.Fatal(err)
	}
	pins := sexpr.PinNumbers(blocks[0])
	if len(pins) != 2 {
		t.Fatalf("expected 2 pins, got %d: %v", len(pins), pins)
	}
	if pins[0] != "1" || pins[1] != "2" {
		t.Errorf("pin numbers = %v, want [1 2]", pins)
	}
}

func TestSerializeTokens_RoundTrip(t *testing.T) {
	blocks, err := sexpr.ExtractSymbolBlocks([]byte(minimalSymLib))
	if err != nil {
		t.Fatal(err)
	}
	serialised := sexpr.SerializeTokens(blocks[0])

	rewrapped := "(kicad_symbol_lib\n" + serialised + "\n)"
	blocks2, err := sexpr.ExtractSymbolBlocks([]byte(rewrapped))
	if err != nil {
		t.Fatalf("could not re-parse serialised block: %v\n%s", err, serialised)
	}
	if len(blocks2) != 1 {
		t.Fatalf("expected 1 block after round-trip, got %d", len(blocks2))
	}
	if sexpr.BlockName(blocks2[0]) != "TestPart" {
		t.Errorf("BlockName after round-trip = %q", sexpr.BlockName(blocks2[0]))
	}
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\ngot:\n%s", needle, haystack)
	}
}
