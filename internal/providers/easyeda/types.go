package easyeda

import "encoding/json"

// ImportRequest describes a request to import EasyEDA/LCSC assets for a component.
type ImportRequest struct {
	ComponentID string
	LCSCID      string
}

// ImportResult describes the outcome of an EasyEDA/LCSC import.
type ImportResult struct {
	LCSCID            string   `json:"lcscId"`
	SymbolImported    bool     `json:"symbolImported"`
	FootprintImported bool     `json:"footprintImported"`
	Model3DImported   bool     `json:"model3dImported"`
	Warnings          []string `json:"warnings"`
	Errors            []string `json:"errors"`
}

// conversionError describes a typed failure during conversion.
type conversionError struct {
	Phase   string // "symbol", "footprint", "3d_model"
	Message string
	Raw     json.RawMessage // upstream payload excerpt for debugging
}

func (e *conversionError) Error() string {
	return e.Phase + ": " + e.Message
}

// --- EasyEDA symbol raw shape types ---

// eeSymbolBbox is the bounding box origin for the symbol canvas.
type eeSymbolBbox struct {
	X float64
	Y float64
}

// eeSymbolInfo holds metadata extracted from the symbol's c_para.
type eeSymbolInfo struct {
	Name         string
	Prefix       string
	Package      string
	Manufacturer string
	DatasheetURL string
	LCSCID       string
}

// eeSymbolPin holds a parsed symbol pin.
type eeSymbolPin struct {
	Name       string
	Number     string
	PosX       float64
	PosY       float64
	Rotation   float64
	Length     float64
	PinType    int // 0=unspecified, 1=input, 2=output, 3=bidirectional, 4=power
	IsInverted bool
	IsClock    bool
}

// eeSymbolRectangle holds a parsed rectangle shape.
type eeSymbolRectangle struct {
	PosX   float64
	PosY   float64
	Width  float64
	Height float64
}

// eeSymbolCircle holds a parsed circle shape.
type eeSymbolCircle struct {
	CenterX float64
	CenterY float64
	Radius  float64
	Filled  bool
}

// eeSymbolPolyline holds a parsed polyline or polygon.
type eeSymbolPolyline struct {
	Points []float64 // alternating x, y
	Closed bool
}

// --- EasyEDA footprint raw shape types ---

// eeFootprintBbox is the bounding box origin for the footprint canvas.
type eeFootprintBbox struct {
	X float64
	Y float64
}

// eeFootprintInfo holds metadata about the footprint.
type eeFootprintInfo struct {
	Name   string
	FPType string // "smd" or "tht"
}

// eeFootprintPad holds a parsed pad.
type eeFootprintPad struct {
	Shape      string // ELLIPSE, RECT, OVAL, POLYGON
	CenterX    float64
	CenterY    float64
	Width      float64
	Height     float64
	LayerID    int
	Number     string
	HoleRadius float64
	HoleLength float64
	Rotation   float64
	Points     string // raw polygon points if custom shape
}

// eeFootprintTrack holds a parsed track (line segments).
type eeFootprintTrack struct {
	StrokeWidth float64
	LayerID     int
	Points      string // space-separated coordinate pairs
}

// eeFootprintHole holds a parsed mounting hole.
type eeFootprintHole struct {
	CenterX float64
	CenterY float64
	Radius  float64
}

// eeFootprintCircle holds a parsed circle.
type eeFootprintCircle struct {
	CX          float64
	CY          float64
	Radius      float64
	StrokeWidth float64
	LayerID     int
}

// eeFootprintArc holds a raw arc SVG path string.
type eeFootprintArc struct {
	StrokeWidth float64
	LayerID     int
	Path        string // raw SVG path "M ... A ..."
}

// eeFootprintRectangle holds a parsed rectangle.
type eeFootprintRectangle struct {
	X           float64
	Y           float64
	Width       float64
	Height      float64
	StrokeWidth float64
	LayerID     int
}

// ee3DModelInfo holds SVGNODE-extracted 3D model info.
type ee3DModelInfo struct {
	UUID  string
	Title string
}
