package registry

import "componentmanager/internal/domain"

const (
	AttrCapacitanceF  = "capacitance_f"
	AttrVoltageV      = "voltage_v"
	AttrDielectric    = "dielectric"
	AttrCapacitorType = "capacitor_type"
)

var capacitorDefs = []domain.AttributeDefinition{
	{Key: AttrCapacitanceF, Category: domain.CategoryCapacitor, ValueType: domain.ValueTypeNumber, DisplayName: "Capacitance", Unit: strptr("F")},
	{Key: AttrTolerancePercent, Category: domain.CategoryCapacitor, ValueType: domain.ValueTypeNumber, DisplayName: "Tolerance", Unit: strptr("percent")},
	{Key: AttrVoltageV, Category: domain.CategoryCapacitor, ValueType: domain.ValueTypeNumber, DisplayName: "Voltage Rating", Unit: strptr("V")},
	{Key: AttrPackage, Category: domain.CategoryCapacitor, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrDielectric, Category: domain.CategoryCapacitor, ValueType: domain.ValueTypeText, DisplayName: "Dielectric"},
	{Key: AttrCapacitorType, Category: domain.CategoryCapacitor, ValueType: domain.ValueTypeText, DisplayName: "Capacitor Type"},
}
