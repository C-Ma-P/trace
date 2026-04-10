package registry

import "trace/internal/domain"

const (
	AttrResistanceOhms = "resistance_ohms"
	AttrPowerW         = "power_w"
	AttrTempCoPPMC     = "tempco_ppm_c"
	AttrResistorType   = "resistor_type"
)

var resistorDefs = []domain.AttributeDefinition{
	{Key: AttrResistanceOhms, Category: domain.CategoryResistor, ValueType: domain.ValueTypeNumber, DisplayName: "Resistance", Unit: strptr("ohm")},
	{Key: AttrTolerancePercent, Category: domain.CategoryResistor, ValueType: domain.ValueTypeNumber, DisplayName: "Tolerance", Unit: strptr("percent")},
	{Key: AttrPowerW, Category: domain.CategoryResistor, ValueType: domain.ValueTypeNumber, DisplayName: "Power Rating", Unit: strptr("W")},
	{Key: AttrPackage, Category: domain.CategoryResistor, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrTempCoPPMC, Category: domain.CategoryResistor, ValueType: domain.ValueTypeNumber, DisplayName: "Temperature Coefficient", Unit: strptr("ppm/C")},
	{Key: AttrResistorType, Category: domain.CategoryResistor, ValueType: domain.ValueTypeText, DisplayName: "Resistor Type"},
}
