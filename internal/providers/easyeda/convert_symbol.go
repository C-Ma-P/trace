package easyeda

import (
	"fmt"
	"math"
	"strings"
)

// pxToMM converts EasyEDA symbol pixel units to mm (KiCad v6+).
func pxToMM(dim float64) float64 {
	return 10.0 * dim * 0.0254
}

// kiPinType maps EasyEDA pin type integers to KiCad pin type names.
var kiPinType = map[int]string{
	0: "unspecified",
	1: "input",
	2: "output",
	3: "bidirectional",
	4: "power_in",
}

// kiPinStyle returns the KiCad pin style string.
func kiPinStyle(inverted, clock bool) string {
	if inverted && clock {
		return "inverted_clock"
	}
	if inverted {
		return "inverted"
	}
	if clock {
		return "clock"
	}
	return "line"
}

// convertSymbol generates a KiCad v6 .kicad_sym file from parsed EasyEDA symbol data.
func convertSymbol(ps *parsedSymbol, lcscID string) (string, error) {
	if len(ps.pins) == 0 && len(ps.rectangles) == 0 {
		return "", &conversionError{Phase: "symbol", Message: "no pins or graphics found in symbol data"}
	}

	name := sanitizeSymbolName(ps.info.Name)
	if name == "" {
		name = lcscID
	}

	var b strings.Builder
	b.WriteString("(kicad_symbol_lib\n")
	b.WriteString("  (version 20220914)\n")
	b.WriteString("  (generator \"trace-easyeda-import\")\n")

	// Symbol definition
	fmt.Fprintf(&b, "  (symbol %q\n", name)
	b.WriteString("    (in_bom yes)\n")
	b.WriteString("    (on_board yes)\n")

	// Properties
	yLow, yHigh := pinYBounds(ps.pins)
	fieldOffset := 5.08

	writeProperty(&b, "Reference", ps.info.Prefix, 0, yHigh+fieldOffset, false, 4)
	writeProperty(&b, "Value", name, 1, yLow-fieldOffset, false, 4)

	propID := 2
	if ps.info.Package != "" {
		fieldOffset += 2.54
		writeProperty(&b, "Footprint", ps.info.Package, propID, yLow-fieldOffset, true, 4)
		propID++
	}

	if ps.info.Manufacturer != "" {
		fieldOffset += 2.54
		writeProperty(&b, "Manufacturer", ps.info.Manufacturer, propID, yLow-fieldOffset, true, 4)
		propID++
	}

	if lcscID != "" {
		fieldOffset += 2.54
		writeProperty(&b, "LCSC Part", lcscID, propID, yLow-fieldOffset, true, 4)
		propID++
	}

	// Graphic items sub-symbol
	fmt.Fprintf(&b, "    (symbol %q\n", name+"_0_1")

	// Rectangles
	for _, r := range ps.rectangles {
		x0 := pxToMM(r.PosX)
		y0 := -pxToMM(r.PosY)
		x1 := x0 + pxToMM(r.Width)
		y1 := y0 - pxToMM(r.Height)
		fmt.Fprintf(&b, "      (rectangle\n")
		fmt.Fprintf(&b, "        (start %.2f %.2f)\n", x0, y0)
		fmt.Fprintf(&b, "        (end %.2f %.2f)\n", x1, y1)
		fmt.Fprintf(&b, "        (stroke (width 0) (type default) (color 0 0 0 0))\n")
		fmt.Fprintf(&b, "        (fill (type background))\n")
		fmt.Fprintf(&b, "      )\n")
	}

	// Circles
	for _, c := range ps.circles {
		cx := pxToMM(c.CenterX)
		cy := -pxToMM(c.CenterY)
		r := pxToMM(c.Radius)
		fill := "none"
		if c.Filled {
			fill = "background"
		}
		fmt.Fprintf(&b, "      (circle\n")
		fmt.Fprintf(&b, "        (center %.2f %.2f)\n", cx, cy)
		fmt.Fprintf(&b, "        (radius %.2f)\n", r)
		fmt.Fprintf(&b, "        (stroke (width 0) (type default) (color 0 0 0 0))\n")
		fmt.Fprintf(&b, "        (fill (type %s))\n", fill)
		fmt.Fprintf(&b, "      )\n")
	}

	// Polylines
	for _, pl := range ps.polylines {
		pts := pl.Points
		if pl.Closed && len(pts) >= 4 {
			// Close the polygon by appending the first point
			pts = append(pts, pts[0], pts[1])
		}
		fill := "none"
		if pl.Closed {
			fill = "background"
		}
		fmt.Fprintf(&b, "      (polyline\n")
		fmt.Fprintf(&b, "        (pts\n")
		for i := 0; i < len(pts)-1; i += 2 {
			x := pxToMM(pts[i])
			y := -pxToMM(pts[i+1])
			fmt.Fprintf(&b, "          (xy %.2f %.2f)\n", x, y)
		}
		fmt.Fprintf(&b, "        )\n")
		fmt.Fprintf(&b, "        (stroke (width 0) (type default) (color 0 0 0 0))\n")
		fmt.Fprintf(&b, "        (fill (type %s))\n", fill)
		fmt.Fprintf(&b, "      )\n")
	}

	// Pins
	for _, pin := range ps.pins {
		pinTypeName := kiPinType[pin.PinType]
		if pinTypeName == "" {
			pinTypeName = "unspecified"
		}
		style := kiPinStyle(pin.IsInverted, pin.IsClock)
		x := pxToMM(pin.PosX)
		y := -pxToMM(pin.PosY)
		length := pxToMM(pin.Length)
		orientation := int(math.Mod(180+pin.Rotation, 360))

		pinName := pin.Name
		if pinName == "" {
			pinName = "~"
		}
		pinNum := pin.Number
		if pinNum == "" {
			pinNum = "~"
		}

		fmt.Fprintf(&b, "      (pin %s %s\n", pinTypeName, style)
		fmt.Fprintf(&b, "        (at %.2f %.2f %d)\n", x, y, orientation)
		fmt.Fprintf(&b, "        (length %.2f)\n", length)
		fmt.Fprintf(&b, "        (name %q (effects (font (size 1.27 1.27))))\n", pinName)
		fmt.Fprintf(&b, "        (number %q (effects (font (size 1.27 1.27))))\n", pinNum)
		fmt.Fprintf(&b, "      )\n")
	}

	b.WriteString("    )\n") // close sub-symbol
	b.WriteString("  )\n")   // close symbol
	b.WriteString(")\n")     // close lib

	return b.String(), nil
}

// pinYBounds returns the min and max Y positions of pins in mm, for field placement.
func pinYBounds(pins []eeSymbolPin) (yLow, yHigh float64) {
	if len(pins) == 0 {
		return 0, 0
	}
	yLow = math.MaxFloat64
	yHigh = -math.MaxFloat64
	for _, p := range pins {
		y := -pxToMM(p.PosY) // negate for KiCad coordinate system
		if y < yLow {
			yLow = y
		}
		if y > yHigh {
			yHigh = y
		}
	}
	return yLow, yHigh
}

// writeProperty writes a KiCad v6 property s-expression.
func writeProperty(b *strings.Builder, key, value string, id int, posY float64, hidden bool, indent int) {
	prefix := strings.Repeat("  ", indent/2)
	hide := ""
	if hidden {
		hide = " hide"
	}
	fmt.Fprintf(b, "%s(property %q %q\n", prefix, key, value)
	fmt.Fprintf(b, "%s  (at 0 %.2f 0)\n", prefix, posY)
	fmt.Fprintf(b, "%s  (effects (font (size 1.27 1.27))%s)\n", prefix, hide)
	fmt.Fprintf(b, "%s)\n", prefix)
}

// sanitizeSymbolName cleans up a name for use as a KiCad symbol identifier.
func sanitizeSymbolName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	return name
}
