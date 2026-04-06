package registry

// Keys shared across multiple categories.
const (
	AttrManufacturer     = "manufacturer"
	AttrMPN              = "mpn"
	AttrPackage          = "package"
	AttrTolerancePercent = "tolerance_percent"

	// AttrLCSCPart is the canonical attribute key for a component's LCSC part number (e.g. "C2040").
	// Setting this attribute enables one-click EasyEDA/LCSC asset import.
	AttrLCSCPart = "lcsc_part"
)
