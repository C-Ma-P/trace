package registry

import "github.com/C-Ma-P/trace/internal/domain"

const (
	AttrDiodeType       = "diode_type"
	AttrReverseVoltageV = "reverse_voltage_v"
	AttrForwardCurrentA = "forward_current_a"
	AttrForwardVoltageV = "forward_voltage_v"
	AttrLEDColor        = "led_color"
)

var diodeDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryDiode, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrDiodeType, Category: domain.CategoryDiode, ValueType: domain.ValueTypeText, DisplayName: "Diode Type"},
	{Key: AttrReverseVoltageV, Category: domain.CategoryDiode, ValueType: domain.ValueTypeNumber, DisplayName: "Reverse Voltage", Unit: strptr("V")},
	{Key: AttrForwardCurrentA, Category: domain.CategoryDiode, ValueType: domain.ValueTypeNumber, DisplayName: "Forward Current", Unit: strptr("A")},
	{Key: AttrForwardVoltageV, Category: domain.CategoryDiode, ValueType: domain.ValueTypeNumber, DisplayName: "Forward Voltage", Unit: strptr("V")},
}

var ledDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryLED, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrLEDColor, Category: domain.CategoryLED, ValueType: domain.ValueTypeText, DisplayName: "Colour"},
	{Key: AttrForwardVoltageV, Category: domain.CategoryLED, ValueType: domain.ValueTypeNumber, DisplayName: "Forward Voltage", Unit: strptr("V")},
	{Key: AttrForwardCurrentA, Category: domain.CategoryLED, ValueType: domain.ValueTypeNumber, DisplayName: "Forward Current", Unit: strptr("A")},
}
