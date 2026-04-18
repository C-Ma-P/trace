package sch

import (
	"fmt"
	"io"
	"strings"

	"github.com/C-Ma-P/trace/internal/kicad"
	"github.com/C-Ma-P/trace/internal/kicad/sexpr"
	"github.com/C-Ma-P/trace/internal/kicad/sym"
)

const (
	schVersion          = "20250114"
	schGenerator        = "trace-export"
	schGeneratorVersion = "9.0"
	libName             = "trace_export"

	paperSize = "A3"
)

func Write(w io.Writer, projectName, schUUID string, parts []kicad.ExportedPart, lib *sym.Library) error {
	out := sexpr.New()
	out.Enter("kicad_sch")
	out.Line(fmt.Sprintf("(version %s)", schVersion))
	out.Line(fmt.Sprintf("(generator %s)", sexpr.Q(schGenerator)))
	out.Line(fmt.Sprintf("(generator_version %s)", sexpr.Q(schGeneratorVersion)))
	out.Line(fmt.Sprintf("(uuid %s)", sexpr.Q(schUUID)))
	out.Line(fmt.Sprintf("(paper %s)", sexpr.Q(paperSize)))

	var libBuf strings.Builder
	if err := lib.WriteLibSymbolsTo(&libBuf, libName); err != nil {
		return fmt.Errorf("write lib_symbols: %w", err)
	}
	if libBuf.Len() > 0 {
		out.Enter("lib_symbols")
		out.Raw(libBuf.String())
		out.Leave()
	}

	for _, p := range parts {
		writeSymbolInstance(out, p, projectName, schUUID)
	}

	out.Enter("sheet_instances")
	out.Enter("path", sexpr.Q("/"))
	out.Line(fmt.Sprintf("(page %s)", sexpr.Q("1")))
	out.Leave()
	out.Leave()

	out.Leave()
	_, err := io.WriteString(w, out.String())
	return err
}

func writeSymbolInstance(out *sexpr.W, p kicad.ExportedPart, projectName, schUUID string) {
	libID := libName + ":" + p.SymbolLibKey
	posX := sexpr.FmtNum(p.X)
	posY := sexpr.FmtNum(p.Y)

	inBOM := boolWord(p.InBOM)
	onBoard := boolWord(p.OnBoard)

	out.Enter("symbol")
	out.Line(fmt.Sprintf("(lib_id %s)", sexpr.Q(libID)))
	out.Line(fmt.Sprintf("(at %s %s 0)", posX, posY))
	out.Line("(unit 1)")
	out.Line(fmt.Sprintf("(in_bom %s)", inBOM))
	out.Line(fmt.Sprintf("(on_board %s)", onBoard))
	out.Line(fmt.Sprintf("(uuid %s)", sexpr.Q(p.UUID)))

	writeProperty(out, "Reference", p.Reference, p.X-1.27, p.Y-2.54, false)
	writeProperty(out, "Value", p.Value, p.X, p.Y+2.54, false)
	writeProperty(out, "Footprint", p.FootprintRef, p.X, p.Y, true)
	ds := p.Datasheet
	if ds == "" {
		ds = "~"
	}
	writeProperty(out, "Datasheet", ds, p.X, p.Y, true)
	writeProperty(out, "Description", "", p.X, p.Y, true)

	if p.MPN != "" {
		writeProperty(out, "MPN", p.MPN, p.X, p.Y, true)
	}
	if p.Manufacturer != "" {
		writeProperty(out, "Manufacturer", p.Manufacturer, p.X, p.Y, true)
	}
	if p.Package != "" {
		writeProperty(out, "Package", p.Package, p.X, p.Y, true)
	}

	out.Enter("instances")
	out.Enter("project", sexpr.Q(safeProjectName(projectName)))
	out.Enter("path", sexpr.Q("/"+schUUID))
	out.Line(fmt.Sprintf("(reference %s)", sexpr.Q(p.Reference)))
	out.Line("(unit 1)")
	out.Leave()
	out.Leave()
	out.Leave()

	out.Leave()
}

func writeProperty(out *sexpr.W, key, value string, x, y float64, hidden bool) {
	out.Enter("property", sexpr.Q(key), sexpr.Q(value))
	out.Line(fmt.Sprintf("(at %s %s 0)", sexpr.FmtNum(x), sexpr.FmtNum(y)))
	out.Enter("effects")
	out.Enter("font")
	out.Line("(size 1.27 1.27)")
	out.Leave()
	if hidden {
		out.Line("(hide yes)")
	}
	out.Leave()
	out.Leave()
}

func boolWord(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func safeProjectName(name string) string {
	var sb strings.Builder
	for _, r := range name {
		if r == ' ' || r == '\t' {
			sb.WriteByte('_')
		} else if r >= 32 && r < 127 && r != '"' && r != '\\' {
			sb.WriteRune(r)
		}
	}
	result := sb.String()
	if result == "" {
		return "project"
	}
	return result
}
