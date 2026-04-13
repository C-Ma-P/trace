package registry

import "github.com/C-Ma-P/trace/internal/domain"

const (
	AttrOutputVoltageV   = "output_voltage_v"
	AttrOutputCurrentA   = "output_current_a"
	AttrDropoutV         = "dropout_v"
	AttrPolarity         = "polarity"
	AttrTopology         = "topology"
	AttrInputVoltageMinV = "input_voltage_min_v"
	AttrInputVoltageMaxV = "input_voltage_max_v"
)

var regulatorLinearDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryRegulatorLinear, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrOutputVoltageV, Category: domain.CategoryRegulatorLinear, ValueType: domain.ValueTypeNumber, DisplayName: "Output Voltage", Unit: strptr("V")},
	{Key: AttrOutputCurrentA, Category: domain.CategoryRegulatorLinear, ValueType: domain.ValueTypeNumber, DisplayName: "Output Current", Unit: strptr("A")},
	{Key: AttrDropoutV, Category: domain.CategoryRegulatorLinear, ValueType: domain.ValueTypeNumber, DisplayName: "Dropout Voltage", Unit: strptr("V")},
	{Key: AttrPolarity, Category: domain.CategoryRegulatorLinear, ValueType: domain.ValueTypeText, DisplayName: "Polarity"},
}

var regulatorSwitchingDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryRegulatorSwitching, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrTopology, Category: domain.CategoryRegulatorSwitching, ValueType: domain.ValueTypeText, DisplayName: "Topology"},
	{Key: AttrOutputVoltageV, Category: domain.CategoryRegulatorSwitching, ValueType: domain.ValueTypeNumber, DisplayName: "Output Voltage", Unit: strptr("V")},
	{Key: AttrOutputCurrentA, Category: domain.CategoryRegulatorSwitching, ValueType: domain.ValueTypeNumber, DisplayName: "Output Current", Unit: strptr("A")},
	{Key: AttrInputVoltageMinV, Category: domain.CategoryRegulatorSwitching, ValueType: domain.ValueTypeNumber, DisplayName: "Input Voltage Min", Unit: strptr("V")},
	{Key: AttrInputVoltageMaxV, Category: domain.CategoryRegulatorSwitching, ValueType: domain.ValueTypeNumber, DisplayName: "Input Voltage Max", Unit: strptr("V")},
}
