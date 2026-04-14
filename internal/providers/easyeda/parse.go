package easyeda

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// --- Symbol parsing ---

// parseSymbolShapes parses the raw EasyEDA symbol dataStr JSON into typed shapes.
func parseSymbolShapes(dataStrRaw json.RawMessage) (*parsedSymbol, error) {
	var dataStr struct {
		Head struct {
			X     json.Number            `json:"x"`
			Y     json.Number            `json:"y"`
			CPara map[string]interface{} `json:"c_para"`
		} `json:"head"`
		Shape []string `json:"shape"`
	}
	if err := json.Unmarshal(dataStrRaw, &dataStr); err != nil {
		return nil, fmt.Errorf("parsing symbol dataStr: %w", err)
	}

	bboxX, _ := dataStr.Head.X.Float64()
	bboxY, _ := dataStr.Head.Y.Float64()

	cpara := dataStr.Head.CPara
	info := eeSymbolInfo{
		Name:         stringFromMap(cpara, "name"),
		Prefix:       strings.TrimSuffix(stringFromMap(cpara, "pre"), "?"),
		Package:      stringFromMap(cpara, "package"),
		Manufacturer: stringFromMap(cpara, "BOM_Manufacturer"),
	}

	ps := &parsedSymbol{
		bbox: eeSymbolBbox{X: bboxX, Y: bboxY},
		info: info,
	}

	for _, line := range dataStr.Shape {
		parts := strings.SplitN(line, "~", 2)
		if len(parts) < 2 {
			continue
		}
		designator := parts[0]
		switch designator {
		case "P":
			if pin, err := parseSymbolPin(line, ps.bbox); err == nil {
				ps.pins = append(ps.pins, pin)
			}
		case "R":
			if rect, err := parseSymbolRectangle(line, ps.bbox); err == nil {
				ps.rectangles = append(ps.rectangles, rect)
			}
		case "C":
			if circ, err := parseSymbolCircle(line, ps.bbox); err == nil {
				ps.circles = append(ps.circles, circ)
			}
		case "PL":
			if pl, err := parseSymbolPolyline(line, ps.bbox, false); err == nil {
				ps.polylines = append(ps.polylines, pl)
			}
		case "PG":
			if pg, err := parseSymbolPolyline(line, ps.bbox, true); err == nil {
				ps.polylines = append(ps.polylines, pg)
			}
		case "E":
			if circ, err := parseSymbolEllipse(line, ps.bbox); err == nil {
				ps.circles = append(ps.circles, circ)
			}
		}
		// A (arcs), PT (paths) skipped in v1 — decorative
	}

	return ps, nil
}

type parsedSymbol struct {
	bbox       eeSymbolBbox
	info       eeSymbolInfo
	pins       []eeSymbolPin
	rectangles []eeSymbolRectangle
	circles    []eeSymbolCircle
	polylines  []eeSymbolPolyline
}

// parseSymbolPin parses a P~...^^...^^... pin entry.
func parseSymbolPin(line string, bbox eeSymbolBbox) (eeSymbolPin, error) {
	segments := strings.Split(line, "^^")
	if len(segments) < 7 {
		return eeSymbolPin{}, fmt.Errorf("pin: expected >=7 segments, got %d", len(segments))
	}

	// Segment 0: settings (tilde-separated, first element is "P")
	settingFields := strings.Split(segments[0], "~")
	if len(settingFields) < 7 {
		return eeSymbolPin{}, fmt.Errorf("pin settings: expected >=7 fields, got %d", len(settingFields))
	}

	// Fields from settings (after "P"): is_displayed, type, spice_pin_number,
	// pos_x, pos_y, rotation, id, is_locked
	pinType := safeInt(settingFields[2])
	posX := safeFloat(settingFields[4])
	posY := safeFloat(settingFields[5])
	rotation := safeFloat(settingFields[6])
	spicePinNum := settingFields[3]

	// Segment 2: pin path + color — extract length from "M x y h LENGTH"
	pathFields := strings.Split(segments[2], "~")
	pinLength := 0.0
	if len(pathFields) >= 1 {
		path := strings.ReplaceAll(pathFields[0], "v", "h")
		hParts := strings.Split(path, "h")
		if len(hParts) >= 2 {
			lastH := strings.TrimSpace(hParts[len(hParts)-1])
			if v, err := strconv.ParseFloat(lastH, 64); err == nil {
				pinLength = math.Abs(v)
			}
		}
	}

	// Segment 3: pin name
	nameFields := strings.Split(segments[3], "~")
	pinName := ""
	if len(nameFields) >= 5 {
		pinName = strings.ReplaceAll(nameFields[4], " ", "")
	}

	// Segment 5: inverted dot
	dotFields := strings.Split(segments[5], "~")
	isInverted := len(dotFields) >= 1 && dotFields[0] == "show"

	// Segment 6: clock
	clockFields := strings.Split(segments[6], "~")
	isClock := len(clockFields) >= 1 && clockFields[0] == "show"

	return eeSymbolPin{
		Name:       pinName,
		Number:     strings.ReplaceAll(spicePinNum, " ", ""),
		PosX:       posX - bbox.X,
		PosY:       posY - bbox.Y,
		Rotation:   rotation,
		Length:     pinLength,
		PinType:    pinType,
		IsInverted: isInverted,
		IsClock:    isClock,
	}, nil
}

// parseSymbolRectangle parses an R~... rectangle entry.
func parseSymbolRectangle(line string, bbox eeSymbolBbox) (eeSymbolRectangle, error) {
	fields := strings.Split(line, "~")
	if len(fields) < 8 {
		return eeSymbolRectangle{}, fmt.Errorf("rectangle: expected >=8 fields, got %d", len(fields))
	}
	return eeSymbolRectangle{
		PosX:   safeFloat(fields[1]) - bbox.X,
		PosY:   safeFloat(fields[2]) - bbox.Y,
		Width:  safeFloat(fields[5]),
		Height: safeFloat(fields[6]),
	}, nil
}

// parseSymbolCircle parses a C~... circle entry.
func parseSymbolCircle(line string, bbox eeSymbolBbox) (eeSymbolCircle, error) {
	fields := strings.Split(line, "~")
	if len(fields) < 6 {
		return eeSymbolCircle{}, fmt.Errorf("circle: expected >=6 fields, got %d", len(fields))
	}
	fillColor := ""
	if len(fields) >= 10 {
		fillColor = fields[9]
	}
	return eeSymbolCircle{
		CenterX: safeFloat(fields[1]) - bbox.X,
		CenterY: safeFloat(fields[2]) - bbox.Y,
		Radius:  safeFloat(fields[3]),
		Filled:  fillColor != "" && strings.ToLower(fillColor) != "none",
	}, nil
}

// parseSymbolEllipse parses an E~... ellipse entry.
// KiCad doesn't support ellipses; if radii match, treat as circle.
func parseSymbolEllipse(line string, bbox eeSymbolBbox) (eeSymbolCircle, error) {
	fields := strings.Split(line, "~")
	if len(fields) < 6 {
		return eeSymbolCircle{}, fmt.Errorf("ellipse: expected >=6 fields, got %d", len(fields))
	}
	rx := safeFloat(fields[3])
	ry := safeFloat(fields[4])
	if rx != ry {
		return eeSymbolCircle{}, fmt.Errorf("ellipse: non-circular ellipse not supported (rx=%.1f, ry=%.1f)", rx, ry)
	}
	return eeSymbolCircle{
		CenterX: safeFloat(fields[1]) - bbox.X,
		CenterY: safeFloat(fields[2]) - bbox.Y,
		Radius:  rx,
	}, nil
}

// parseSymbolPolyline parses a PL~... or PG~... polyline/polygon entry.
func parseSymbolPolyline(line string, bbox eeSymbolBbox, isPolygon bool) (eeSymbolPolyline, error) {
	fields := strings.Split(line, "~")
	if len(fields) < 2 {
		return eeSymbolPolyline{}, fmt.Errorf("polyline: not enough fields")
	}
	rawPts := strings.TrimSpace(fields[1])
	parts := strings.Fields(rawPts)

	var points []float64
	for i := 0; i < len(parts)-1; i += 2 {
		x := safeFloat(parts[i]) - bbox.X
		y := safeFloat(parts[i+1]) - bbox.Y
		points = append(points, x, y)
	}
	if len(points) < 4 {
		return eeSymbolPolyline{}, fmt.Errorf("polyline: too few points")
	}

	// Check fill color for closure
	closed := isPolygon
	if len(fields) >= 5 {
		fillColor := fields[4]
		if fillColor != "" && strings.ToLower(fillColor) != "none" {
			closed = true
		}
	}

	return eeSymbolPolyline{
		Points: points,
		Closed: closed,
	}, nil
}

// --- Footprint parsing ---

type parsedFootprint struct {
	bbox       eeFootprintBbox
	info       eeFootprintInfo
	pads       []eeFootprintPad
	tracks     []eeFootprintTrack
	holes      []eeFootprintHole
	circles    []eeFootprintCircle
	arcs       []eeFootprintArc
	rectangles []eeFootprintRectangle
	model3D    *ee3DModelInfo
}

// parseFootprintShapes parses the raw EasyEDA footprint packageDetail JSON.
func parseFootprintShapes(pkgDetailRaw json.RawMessage, isSMD bool) (*parsedFootprint, error) {
	var pkgDetail struct {
		Title   string `json:"title"`
		DataStr struct {
			Head struct {
				X     json.Number            `json:"x"`
				Y     json.Number            `json:"y"`
				CPara map[string]interface{} `json:"c_para"`
			} `json:"head"`
			Shape []string `json:"shape"`
		} `json:"dataStr"`
	}
	if err := json.Unmarshal(pkgDetailRaw, &pkgDetail); err != nil {
		return nil, fmt.Errorf("parsing footprint packageDetail: %w", err)
	}

	bboxX, _ := pkgDetail.DataStr.Head.X.Float64()
	bboxY, _ := pkgDetail.DataStr.Head.Y.Float64()

	fpType := "tht"
	if isSMD && !strings.Contains(pkgDetail.Title, "-TH_") {
		fpType = "smd"
	}

	pkgName := stringFromMap(pkgDetail.DataStr.Head.CPara, "package")
	if pkgName == "" {
		pkgName = pkgDetail.Title
	}

	pf := &parsedFootprint{
		bbox: eeFootprintBbox{X: bboxX, Y: bboxY},
		info: eeFootprintInfo{Name: pkgName, FPType: fpType},
	}

	for _, line := range pkgDetail.DataStr.Shape {
		parts := strings.SplitN(line, "~", 2)
		if len(parts) < 2 {
			continue
		}
		designator := parts[0]
		fields := strings.Split(parts[1], "~")

		switch designator {
		case "PAD":
			if pad, err := parseFootprintPad(fields); err == nil {
				pf.pads = append(pf.pads, pad)
			}
		case "TRACK":
			if track, err := parseFootprintTrack(fields); err == nil {
				pf.tracks = append(pf.tracks, track)
			}
		case "HOLE":
			if hole, err := parseFootprintHole(fields); err == nil {
				pf.holes = append(pf.holes, hole)
			}
		case "CIRCLE":
			if circ, err := parseFootprintCircle(fields); err == nil {
				pf.circles = append(pf.circles, circ)
			}
		case "ARC":
			if arc, err := parseFootprintArc(fields); err == nil {
				pf.arcs = append(pf.arcs, arc)
			}
		case "RECT":
			if rect, err := parseFootprintRect(fields); err == nil {
				pf.rectangles = append(pf.rectangles, rect)
			}
		case "SVGNODE":
			if len(fields) >= 1 {
				if model := parse3DModelInfo(fields[0]); model != nil {
					pf.model3D = model
				}
			}
		}
	}

	return pf, nil
}

func parseFootprintPad(fields []string) (eeFootprintPad, error) {
	if len(fields) < 13 {
		return eeFootprintPad{}, fmt.Errorf("pad: expected >=13 fields, got %d", len(fields))
	}
	number := fields[7]
	// Clean up number: some have "(1)" format
	if strings.Contains(number, "(") && strings.Contains(number, ")") {
		start := strings.Index(number, "(")
		end := strings.Index(number, ")")
		if start < end {
			number = number[start+1 : end]
		}
	}
	return eeFootprintPad{
		Shape:      fields[0],
		CenterX:    safeFloat(fields[1]),
		CenterY:    safeFloat(fields[2]),
		Width:      safeFloat(fields[3]),
		Height:     safeFloat(fields[4]),
		LayerID:    safeInt(fields[5]),
		Number:     number,
		HoleRadius: safeFloat(fields[8]),
		Points:     safeString(fields, 9),
		Rotation:   safeFloat(safeString(fields, 10)),
		HoleLength: safeFloat(safeString(fields, 12)),
	}, nil
}

func parseFootprintTrack(fields []string) (eeFootprintTrack, error) {
	if len(fields) < 4 {
		return eeFootprintTrack{}, fmt.Errorf("track: expected >=4 fields, got %d", len(fields))
	}
	return eeFootprintTrack{
		StrokeWidth: safeFloat(fields[0]),
		LayerID:     safeInt(fields[1]),
		Points:      fields[3],
	}, nil
}

func parseFootprintHole(fields []string) (eeFootprintHole, error) {
	if len(fields) < 3 {
		return eeFootprintHole{}, fmt.Errorf("hole: expected >=3 fields, got %d", len(fields))
	}
	return eeFootprintHole{
		CenterX: safeFloat(fields[0]),
		CenterY: safeFloat(fields[1]),
		Radius:  safeFloat(fields[2]),
	}, nil
}

func parseFootprintCircle(fields []string) (eeFootprintCircle, error) {
	if len(fields) < 6 {
		return eeFootprintCircle{}, fmt.Errorf("circle: expected >=6 fields, got %d", len(fields))
	}
	return eeFootprintCircle{
		CX:          safeFloat(fields[0]),
		CY:          safeFloat(fields[1]),
		Radius:      safeFloat(fields[2]),
		StrokeWidth: safeFloat(fields[3]),
		LayerID:     safeInt(fields[4]),
	}, nil
}

func parseFootprintArc(fields []string) (eeFootprintArc, error) {
	if len(fields) < 5 {
		return eeFootprintArc{}, fmt.Errorf("arc: expected >=5 fields, got %d", len(fields))
	}
	return eeFootprintArc{
		StrokeWidth: safeFloat(fields[0]),
		LayerID:     safeInt(fields[1]),
		Path:        fields[3],
	}, nil
}

func parseFootprintRect(fields []string) (eeFootprintRectangle, error) {
	if len(fields) < 6 {
		return eeFootprintRectangle{}, fmt.Errorf("rect: expected >=6 fields, got %d", len(fields))
	}
	return eeFootprintRectangle{
		X:           safeFloat(fields[0]),
		Y:           safeFloat(fields[1]),
		Width:       safeFloat(fields[2]),
		Height:      safeFloat(fields[3]),
		StrokeWidth: safeFloat(fields[4]),
		LayerID:     safeInt(fields[6]),
	}, nil
}

func parse3DModelInfo(jsonPayload string) *ee3DModelInfo {
	var node struct {
		Attrs struct {
			UUID  string `json:"uuid"`
			Title string `json:"title"`
		} `json:"attrs"`
	}
	if json.Unmarshal([]byte(jsonPayload), &node) != nil {
		return nil
	}
	if node.Attrs.UUID == "" {
		return nil
	}
	return &ee3DModelInfo{UUID: node.Attrs.UUID, Title: node.Attrs.Title}
}

// --- Helpers ---

func safeFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

func safeInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

func safeString(fields []string, idx int) string {
	if idx < len(fields) {
		return fields[idx]
	}
	return ""
}

func stringFromMap(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		return ""
	}
	return s
}
