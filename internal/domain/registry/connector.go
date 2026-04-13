package registry

import "github.com/C-Ma-P/trace/internal/domain"

const (
	AttrConnectorType = "connector_type"
	AttrPositions     = "positions"
	AttrPitchMM       = "pitch_mm"
	AttrMountingType  = "mounting_type"
	AttrSwitchType    = "switch_type"
	AttrPoles         = "poles"
	AttrThrows        = "throws"
)

var connectorDefs = []domain.AttributeDefinition{
	{Key: AttrConnectorType, Category: domain.CategoryConnector, ValueType: domain.ValueTypeText, DisplayName: "Connector Type"},
	{Key: AttrPositions, Category: domain.CategoryConnector, ValueType: domain.ValueTypeNumber, DisplayName: "Positions"},
	{Key: AttrPitchMM, Category: domain.CategoryConnector, ValueType: domain.ValueTypeNumber, DisplayName: "Pitch", Unit: strptr("mm")},
	{Key: AttrMountingType, Category: domain.CategoryConnector, ValueType: domain.ValueTypeText, DisplayName: "Mounting Type"},
}

var switchDefs = []domain.AttributeDefinition{
	{Key: AttrSwitchType, Category: domain.CategorySwitch, ValueType: domain.ValueTypeText, DisplayName: "Switch Type"},
	{Key: AttrPoles, Category: domain.CategorySwitch, ValueType: domain.ValueTypeNumber, DisplayName: "Poles"},
	{Key: AttrThrows, Category: domain.CategorySwitch, ValueType: domain.ValueTypeNumber, DisplayName: "Throws"},
	{Key: AttrMountingType, Category: domain.CategorySwitch, ValueType: domain.ValueTypeText, DisplayName: "Mounting Type"},
}
