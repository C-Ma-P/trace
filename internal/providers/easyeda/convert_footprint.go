package easyeda

import (
	"fmt"
	"math"
	"strings"
)

// eeToMM converts EasyEDA footprint units to mm.
func eeToMM(dim float64) float64 {
	return dim * 10 * 0.0254
}

// Layer maps for footprint conversion

var kiLayers = map[int]string{
	1: "F.Cu", 2: "B.Cu", 3: "F.SilkS", 4: "B.SilkS",
	5: "F.Paste", 6: "B.Paste", 7: "F.Mask", 8: "B.Mask",
	10: "Edge.Cuts", 11: "Edge.Cuts", 12: "Cmts.User",
	13: "F.Fab", 14: "B.Fab", 15: "Dwgs.User", 101: "F.Fab",
}

var kiPadLayerSMD = map[int]string{
	1: "F.Cu F.Paste F.Mask", 2: "B.Cu B.Paste B.Mask",
	3: "F.SilkS", 11: "*.Cu *.Paste *.Mask", 13: "F.Fab", 15: "Dwgs.User",
}

var kiPadLayerTHT = map[int]string{
	1: "F.Cu F.Mask", 2: "B.Cu B.Mask", 3: "F.SilkS",
	11: "*.Cu *.Mask", 13: "F.Fab", 15: "Dwgs.User",
}

var kiPadShape = map[string]string{
	"ELLIPSE": "circle", "RECT": "rect", "OVAL": "oval", "POLYGON": "custom",
}

// convertFootprint generates a KiCad .kicad_mod file from parsed EasyEDA footprint data.
func convertFootprint(pf *parsedFootprint) (string, error) {
	if len(pf.pads) == 0 {
		return "", &conversionError{Phase: "footprint", Message: "no pads found in footprint data"}
	}

	name := sanitizeFootprintName(pf.info.Name)
	if name == "" {
		return "", &conversionError{Phase: "footprint", Message: "footprint has no name"}
	}

	// Convert bbox to mm
	bboxX := eeToMM(pf.bbox.X)
	bboxY := eeToMM(pf.bbox.Y)

	var b strings.Builder

	// Module header
	fmt.Fprintf(&b, "(module easyeda2trace:%s (layer F.Cu) (tedit 00000000)\n", name)

	// Component type
	fpType := "through_hole"
	if pf.info.FPType == "smd" {
		fpType = "smd"
	}
	fmt.Fprintf(&b, "  (attr %s)\n", fpType)

	// Compute pad Y bounds for reference/value placement
	yLow, yHigh := padYBounds(pf.pads, bboxX, bboxY)

	// Reference and value text
	fmt.Fprintf(&b, "  (fp_text reference REF** (at 0 %.2f) (layer F.SilkS)\n", yLow-4)
	b.WriteString("    (effects (font (size 1 1) (thickness 0.15)))\n")
	b.WriteString("  )\n")
	fmt.Fprintf(&b, "  (fp_text value %s (at 0 %.2f) (layer F.Fab)\n", name, yHigh+4)
	b.WriteString("    (effects (font (size 1 1) (thickness 0.15)))\n")
	b.WriteString("  )\n")
	// Fab user reference
	b.WriteString("  (fp_text user %R (at 0 0) (layer F.Fab)\n")
	b.WriteString("    (effects (font (size 1 1) (thickness 0.15)))\n")
	b.WriteString("  )\n")

	// Tracks (line segments)
	for _, track := range pf.tracks {
		sw := eeToMM(track.StrokeWidth)
		if sw < 0.01 {
			sw = 0.01
		}
		layers := lookupLayer(track.LayerID, kiPadLayerSMD, "F.Fab")
		points := parseTrackPoints(track.Points, bboxX, bboxY)
		for i := 0; i < len(points)-3; i += 2 {
			fmt.Fprintf(&b, "  (fp_line (start %.2f %.2f) (end %.2f %.2f) (layer %s) (width %.2f))\n",
				points[i], points[i+1], points[i+2], points[i+3], layers, sw)
		}
	}

	// Rectangles (drawn as 4 lines)
	for _, rect := range pf.rectangles {
		sw := eeToMM(rect.StrokeWidth)
		if sw < 0.01 {
			sw = 0.01
		}
		layers := lookupLayer(rect.LayerID, kiPadLayerSMD, "F.Fab")
		sx := eeToMM(rect.X) - bboxX
		sy := eeToMM(rect.Y) - bboxY
		w := eeToMM(rect.Width)
		h := eeToMM(rect.Height)

		corners := [4][2]float64{
			{sx, sy}, {sx + w, sy}, {sx + w, sy + h}, {sx, sy + h},
		}
		for i := 0; i < 4; i++ {
			j := (i + 1) % 4
			fmt.Fprintf(&b, "  (fp_line (start %.2f %.2f) (end %.2f %.2f) (layer %s) (width %.2f))\n",
				corners[i][0], corners[i][1], corners[j][0], corners[j][1], layers, sw)
		}
	}

	// Pads
	for _, pad := range pf.pads {
		writePad(&b, pad, bboxX, bboxY)
	}

	// Holes
	for _, hole := range pf.holes {
		cx := eeToMM(hole.CenterX) - bboxX
		cy := eeToMM(hole.CenterY) - bboxY
		size := eeToMM(hole.Radius) * 2
		fmt.Fprintf(&b, "  (pad \"\" thru_hole circle (at %.2f %.2f) (size %.2f %.2f) (drill %.2f) (layers *.Cu *.Mask))\n",
			cx, cy, size, size, size)
	}

	// Circles
	for _, circ := range pf.circles {
		cx := eeToMM(circ.CX) - bboxX
		cy := eeToMM(circ.CY) - bboxY
		r := eeToMM(circ.Radius)
		endX := cx + r
		endY := cy
		sw := eeToMM(circ.StrokeWidth)
		if sw < 0.01 {
			sw = 0.01
		}
		layers := lookupLayer(circ.LayerID, kiLayers, "F.Fab")
		fmt.Fprintf(&b, "  (fp_circle (center %.2f %.2f) (end %.2f %.2f) (layer %s) (width %.2f))\n",
			cx, cy, endX, endY, layers, sw)
	}

	// Arcs
	for _, arc := range pf.arcs {
		writeArc(&b, arc, bboxX, bboxY)
	}

	b.WriteString(")\n")
	return b.String(), nil
}

// writePad writes a single KiCad pad entry.
func writePad(b *strings.Builder, pad eeFootprintPad, bboxX, bboxY float64) {
	cx := eeToMM(pad.CenterX) - bboxX
	cy := eeToMM(pad.CenterY) - bboxY
	w := eeToMM(pad.Width)
	h := eeToMM(pad.Height)
	holeR := eeToMM(pad.HoleRadius)
	holeL := eeToMM(pad.HoleLength)

	if w < 0.01 {
		w = 0.01
	}
	if h < 0.01 {
		h = 0.01
	}

	padType := "smd"
	if holeR > 0 {
		padType = "thru_hole"
	}

	shape := kiPadShape[pad.Shape]
	if shape == "" {
		shape = "rect" // safe default
	}

	var layers string
	if holeR > 0 {
		layers = lookupLayer(pad.LayerID, kiPadLayerTHT, "*.Cu *.Mask")
	} else {
		layers = lookupLayer(pad.LayerID, kiPadLayerSMD, "F.Cu F.Paste F.Mask")
	}

	rotation := angleToKi(pad.Rotation)

	// Custom polygon pad handling
	isCustom := shape == "custom"
	polygon := ""
	if isCustom {
		polyPts := parseCustomPadPolygon(pad.Points, bboxX, bboxY, cx, cy)
		if polyPts != "" {
			w = 0.005
			h = 0.005
			rotation = 0
			polygon = polyPts
		}
	}

	drill := formatDrill(holeR, holeL, h, w)

	fmt.Fprintf(b, "  (pad %q %s %s (at %.2f %.2f", pad.Number, padType, shape, cx, cy)
	if rotation != 0 {
		fmt.Fprintf(b, " %.2f", rotation)
	}
	fmt.Fprintf(b, ") (size %.2f %.2f) (layers %s)", w, h, layers)
	if drill != "" {
		fmt.Fprintf(b, " %s", drill)
	}
	if polygon != "" {
		fmt.Fprintf(b, "%s", polygon)
	}
	b.WriteString(")\n")
}

// formatDrill builds the (drill ...) s-expression.
func formatDrill(holeRadius, holeLength, padHeight, padWidth float64) string {
	if holeRadius <= 0 {
		return ""
	}
	if holeLength > 0 {
		maxDist := math.Max(holeRadius*2, holeLength)
		pos0 := padHeight - maxDist
		pos90 := padWidth - maxDist
		if pos0 >= pos90 {
			return fmt.Sprintf("(drill oval %.2f %.2f)", holeRadius*2, holeLength)
		}
		return fmt.Sprintf("(drill oval %.2f %.2f)", holeLength, holeRadius*2)
	}
	return fmt.Sprintf("(drill %.2f)", 2*holeRadius)
}

// angleToKi converts EasyEDA rotation to KiCad orientation.
func angleToKi(rotation float64) float64 {
	if math.IsNaN(rotation) {
		return 0
	}
	if rotation > 180 {
		return -(360 - rotation)
	}
	return rotation
}

// parseCustomPadPolygon builds custom pad polygon primitives.
func parseCustomPadPolygon(rawPoints string, bboxX, bboxY, padX, padY float64) string {
	rawPoints = strings.TrimSpace(rawPoints)
	if rawPoints == "" {
		return ""
	}
	parts := strings.Fields(rawPoints)
	if len(parts) < 4 {
		return ""
	}
	var pts []string
	for i := 0; i < len(parts)-1; i += 2 {
		px := eeToMM(safeFloat(parts[i])) - bboxX - padX
		py := eeToMM(safeFloat(parts[i+1])) - bboxY - padY
		pts = append(pts, fmt.Sprintf("(xy %.2f %.2f)", px, py))
	}
	return fmt.Sprintf("\n    (primitives \n      (gr_poly \n        (pts %s\n        ) \n        (width 0.1) \n      )\n    )\n  ",
		strings.Join(pts, " "))
}

// writeArc parses and writes an SVG arc path to the footprint.
func writeArc(b *strings.Builder, arc eeFootprintArc, bboxX, bboxY float64) {
	path := strings.ReplaceAll(arc.Path, ",", " ")
	path = strings.ReplaceAll(path, "M ", "M")
	path = strings.ReplaceAll(path, "A ", "A")

	aParts := strings.SplitN(path, "A", 2)
	if len(aParts) < 2 {
		return
	}

	startStr := strings.TrimPrefix(aParts[0], "M")
	startFields := strings.Fields(startStr)
	if len(startFields) < 2 {
		return
	}

	startX := eeToMM(safeFloat(startFields[0])) - bboxX
	startY := eeToMM(safeFloat(startFields[1])) - bboxY

	arcParams := strings.Fields(strings.TrimSpace(aParts[1]))
	if len(arcParams) < 7 {
		return
	}

	rx := eeToMM(safeFloat(arcParams[0]))
	ry := eeToMM(safeFloat(arcParams[1]))
	xAxisRot := safeFloat(arcParams[2])
	largeArc := arcParams[3] == "1"
	sweep := arcParams[4] == "1"
	endX := eeToMM(safeFloat(arcParams[5])) - bboxX
	endY := eeToMM(safeFloat(arcParams[6])) - bboxY

	if ry == 0 {
		return
	}

	cx, cy, extent := computeArc(startX, startY, rx, ry, xAxisRot, largeArc, sweep, endX, endY)

	sw := eeToMM(arc.StrokeWidth)
	if sw < 0.01 {
		sw = 0.01
	}
	layers := lookupLayer(arc.LayerID, kiLayers, "F.Fab")

	fmt.Fprintf(b, "  (fp_arc (start %.2f %.2f) (end %.2f %.2f) (angle %.2f) (layer %s) (width %.2f))\n",
		cx, cy, endX, endY, extent, layers, sw)
}

// computeArc implements the SVG endpoint-to-center arc conversion algorithm.
func computeArc(startX, startY, rx, ry, angle float64, largeArc, sweep bool, endX, endY float64) (cx, cy, extent float64) {
	dx2 := (startX - endX) / 2
	dy2 := (startY - endY) / 2

	rad := math.Mod(angle, 360) * math.Pi / 180
	cosA := math.Cos(rad)
	sinA := math.Sin(rad)

	x1 := cosA*dx2 + sinA*dy2
	y1 := -sinA*dx2 + cosA*dy2

	rx = math.Abs(rx)
	ry = math.Abs(ry)
	rx2 := rx * rx
	ry2 := ry * ry
	x12 := x1 * x1
	y12 := y1 * y1

	// Ensure radii are large enough
	check := 0.0
	if rx2 != 0 && ry2 != 0 {
		check = x12/rx2 + y12/ry2
	}
	if check > 1 {
		s := math.Sqrt(check)
		rx *= s
		ry *= s
		rx2 = rx * rx
		ry2 = ry * ry
	}

	sign := -1.0
	if largeArc != sweep {
		sign = 1.0
	}
	sq := 0.0
	denom := rx2*y12 + ry2*x12
	if denom > 0 {
		num := rx2*ry2 - rx2*y12 - ry2*x12
		sq = math.Max(num/denom, 0)
	}
	coef := sign * math.Sqrt(sq)

	cx1 := coef * (rx * y1 / ry)
	cy1 := 0.0
	if rx != 0 {
		cy1 = coef * -(ry * x1 / rx)
	}

	sx2 := (startX + endX) / 2
	sy2 := (startY + endY) / 2
	cx = sx2 + cosA*cx1 - sinA*cy1
	cy = sy2 + sinA*cx1 + cosA*cy1

	// Compute angle extent
	ux, uy := 0.0, 0.0
	if rx != 0 {
		ux = (x1 - cx1) / rx
	}
	if ry != 0 {
		uy = (y1 - cy1) / ry
	}
	vx, vy := 0.0, 0.0
	if rx != 0 {
		vx = (-x1 - cx1) / rx
	}
	if ry != 0 {
		vy = (-y1 - cy1) / ry
	}

	n := math.Sqrt((ux*ux + uy*uy) * (vx*vx + vy*vy))
	p := ux*vx + uy*vy
	esign := 1.0
	if ux*vy-uy*vx < 0 {
		esign = -1.0
	}

	if n != 0 {
		ratio := p / n
		if math.Abs(ratio) < 1 {
			extent = esign * math.Acos(ratio) * 180 / math.Pi
		} else {
			extent = 360 + 359
		}
	} else {
		extent = 360 + 359
	}

	if !sweep && extent > 0 {
		extent -= 360
	} else if sweep && extent < 0 {
		extent += 360
	}

	s2 := 1.0
	if extent >= 0 {
		s2 = -1.0
	}
	extent = math.Mod(math.Abs(extent), 360) * s2

	return cx, cy, extent
}

// padYBounds returns min and max pad Y positions in mm for text placement.
func padYBounds(pads []eeFootprintPad, bboxX, bboxY float64) (yLow, yHigh float64) {
	if len(pads) == 0 {
		return -2, 2
	}
	yLow = math.MaxFloat64
	yHigh = -math.MaxFloat64
	for _, p := range pads {
		y := eeToMM(p.CenterY) - bboxY
		if y < yLow {
			yLow = y
		}
		if y > yHigh {
			yHigh = y
		}
	}
	return yLow, yHigh
}

// parseTrackPoints parses space-separated coordinate pairs and applies bbox offset + mm conversion.
func parseTrackPoints(rawPoints string, bboxX, bboxY float64) []float64 {
	parts := strings.Fields(rawPoints)
	var out []float64
	for i := 0; i < len(parts)-1; i += 2 {
		x := eeToMM(safeFloat(parts[i])) - bboxX
		y := eeToMM(safeFloat(parts[i+1])) - bboxY
		out = append(out, x, y)
	}
	return out
}

// lookupLayer looks up a KiCad layer name from the given map, falling back to defaultLayer.
func lookupLayer(layerID int, m map[int]string, defaultLayer string) string {
	if s, ok := m[layerID]; ok {
		return s
	}
	if s, ok := kiLayers[layerID]; ok {
		return s
	}
	return defaultLayer
}

// sanitizeFootprintName cleans up a name for use as a KiCad footprint identifier.
func sanitizeFootprintName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	return name
}
