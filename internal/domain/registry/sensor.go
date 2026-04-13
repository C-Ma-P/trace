package registry

import "github.com/C-Ma-P/trace/internal/domain"

const (
	AttrSensorType        = "sensor_type"
	AttrInterfaceType     = "interface_type"
	AttrSupplyVoltageMinV = "supply_voltage_min_v"
	AttrSupplyVoltageMaxV = "supply_voltage_max_v"
	AttrModuleType        = "module_type"
)

var sensorDefs = []domain.AttributeDefinition{
	{Key: AttrSensorType, Category: domain.CategorySensor, ValueType: domain.ValueTypeText, DisplayName: "Sensor Type"},
	{Key: AttrPackage, Category: domain.CategorySensor, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrInterfaceType, Category: domain.CategorySensor, ValueType: domain.ValueTypeText, DisplayName: "Interface"},
	{Key: AttrSupplyVoltageMinV, Category: domain.CategorySensor, ValueType: domain.ValueTypeNumber, DisplayName: "Supply Voltage Min", Unit: strptr("V")},
	{Key: AttrSupplyVoltageMaxV, Category: domain.CategorySensor, ValueType: domain.ValueTypeNumber, DisplayName: "Supply Voltage Max", Unit: strptr("V")},
}

var moduleDefs = []domain.AttributeDefinition{
	{Key: AttrModuleType, Category: domain.CategoryModule, ValueType: domain.ValueTypeText, DisplayName: "Module Type"},
	{Key: AttrSupplyVoltageMinV, Category: domain.CategoryModule, ValueType: domain.ValueTypeNumber, DisplayName: "Supply Voltage Min", Unit: strptr("V")},
	{Key: AttrSupplyVoltageMaxV, Category: domain.CategoryModule, ValueType: domain.ValueTypeNumber, DisplayName: "Supply Voltage Max", Unit: strptr("V")},
}
