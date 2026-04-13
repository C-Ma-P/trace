package registry

import "github.com/C-Ma-P/trace/internal/domain"

const (
	AttrBJTType                  = "bjt_type"
	AttrCollectorEmitterVoltageV = "collector_emitter_voltage_v"
	AttrCollectorCurrentA        = "collector_current_a"
	AttrMOSFETChannel            = "mosfet_channel"
	AttrDrainSourceVoltageV      = "drain_source_voltage_v"
	AttrContinuousDrainCurrentA  = "continuous_drain_current_a"
	AttrRdsOnOhm                 = "rds_on_ohm"
)

var transistorBJTDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryTransistorBJT, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrBJTType, Category: domain.CategoryTransistorBJT, ValueType: domain.ValueTypeText, DisplayName: "BJT Type"},
	{Key: AttrCollectorEmitterVoltageV, Category: domain.CategoryTransistorBJT, ValueType: domain.ValueTypeNumber, DisplayName: "Collector-Emitter Voltage", Unit: strptr("V")},
	{Key: AttrCollectorCurrentA, Category: domain.CategoryTransistorBJT, ValueType: domain.ValueTypeNumber, DisplayName: "Collector Current", Unit: strptr("A")},
	{Key: AttrPowerW, Category: domain.CategoryTransistorBJT, ValueType: domain.ValueTypeNumber, DisplayName: "Power Dissipation", Unit: strptr("W")},
}

var transistorMOSFETDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryTransistorMOSFET, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrMOSFETChannel, Category: domain.CategoryTransistorMOSFET, ValueType: domain.ValueTypeText, DisplayName: "Channel Type"},
	{Key: AttrDrainSourceVoltageV, Category: domain.CategoryTransistorMOSFET, ValueType: domain.ValueTypeNumber, DisplayName: "Drain-Source Voltage", Unit: strptr("V")},
	{Key: AttrContinuousDrainCurrentA, Category: domain.CategoryTransistorMOSFET, ValueType: domain.ValueTypeNumber, DisplayName: "Continuous Drain Current", Unit: strptr("A")},
	{Key: AttrRdsOnOhm, Category: domain.CategoryTransistorMOSFET, ValueType: domain.ValueTypeNumber, DisplayName: "Rds(on)", Unit: strptr("ohm")},
}
