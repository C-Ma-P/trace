package registry

import "github.com/C-Ma-P/trace/internal/domain"

const (
	AttrImpedanceOhms = "impedance_ohms"
	// AttrDCROhm is the DC resistance key for ferrite beads.
	// Distinct from AttrDCROhms ("dcr_ohms") used by inductors.
	AttrDCROhm = "dcr_ohm"
)

var ferriteBeadDefs = []domain.AttributeDefinition{
	{Key: AttrImpedanceOhms, Category: domain.CategoryFerriteBead, ValueType: domain.ValueTypeNumber, DisplayName: "Impedance @ 100MHz", Unit: strptr("ohm")},
	{Key: AttrCurrentA, Category: domain.CategoryFerriteBead, ValueType: domain.ValueTypeNumber, DisplayName: "Rated Current", Unit: strptr("A")},
	{Key: AttrDCROhm, Category: domain.CategoryFerriteBead, ValueType: domain.ValueTypeNumber, DisplayName: "DC Resistance", Unit: strptr("ohm")},
	{Key: AttrPackage, Category: domain.CategoryFerriteBead, ValueType: domain.ValueTypeText, DisplayName: "Package"},
}
