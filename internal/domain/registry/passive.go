package registry

import "github.com/C-Ma-P/trace/internal/domain"

const (
	AttrFrequencyHz       = "frequency_hz"
	AttrLoadCapacitancePF = "load_capacitance_pf"
	AttrTolerancePPM      = "tolerance_ppm"
	AttrFuseType          = "fuse_type"
	AttrBatteryChemistry  = "battery_chemistry"
	AttrNominalVoltageV   = "nominal_voltage_v"
	AttrCapacityAh        = "capacity_ah"
)

var crystalOscillatorDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryCrystalOscillator, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrFrequencyHz, Category: domain.CategoryCrystalOscillator, ValueType: domain.ValueTypeNumber, DisplayName: "Frequency", Unit: strptr("Hz")},
	{Key: AttrLoadCapacitancePF, Category: domain.CategoryCrystalOscillator, ValueType: domain.ValueTypeNumber, DisplayName: "Load Capacitance", Unit: strptr("pF")},
	{Key: AttrTolerancePPM, Category: domain.CategoryCrystalOscillator, ValueType: domain.ValueTypeNumber, DisplayName: "Frequency Tolerance", Unit: strptr("ppm")},
}

var fuseDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryFuse, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrCurrentA, Category: domain.CategoryFuse, ValueType: domain.ValueTypeNumber, DisplayName: "Current Rating", Unit: strptr("A")},
	{Key: AttrVoltageV, Category: domain.CategoryFuse, ValueType: domain.ValueTypeNumber, DisplayName: "Voltage Rating", Unit: strptr("V")},
	{Key: AttrFuseType, Category: domain.CategoryFuse, ValueType: domain.ValueTypeText, DisplayName: "Fuse Type"},
}

var batteryDefs = []domain.AttributeDefinition{
	{Key: AttrBatteryChemistry, Category: domain.CategoryBattery, ValueType: domain.ValueTypeText, DisplayName: "Chemistry"},
	{Key: AttrNominalVoltageV, Category: domain.CategoryBattery, ValueType: domain.ValueTypeNumber, DisplayName: "Nominal Voltage", Unit: strptr("V")},
	{Key: AttrCapacityAh, Category: domain.CategoryBattery, ValueType: domain.ValueTypeNumber, DisplayName: "Capacity", Unit: strptr("Ah")},
	{Key: AttrPackage, Category: domain.CategoryBattery, ValueType: domain.ValueTypeText, DisplayName: "Package"},
}
