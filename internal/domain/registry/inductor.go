package registry

import "github.com/C-Ma-P/trace/internal/domain"

const (
	AttrInductanceH  = "inductance_h"
	AttrCurrentA     = "current_a"
	AttrDCROhms      = "dcr_ohms"
	AttrInductorType = "inductor_type"
)

var inductorDefs = []domain.AttributeDefinition{
	{Key: AttrInductanceH, Category: domain.CategoryInductor, ValueType: domain.ValueTypeNumber, DisplayName: "Inductance", Unit: strptr("H")},
	{Key: AttrTolerancePercent, Category: domain.CategoryInductor, ValueType: domain.ValueTypeNumber, DisplayName: "Tolerance", Unit: strptr("percent")},
	{Key: AttrCurrentA, Category: domain.CategoryInductor, ValueType: domain.ValueTypeNumber, DisplayName: "Rated Current", Unit: strptr("A")},
	{Key: AttrDCROhms, Category: domain.CategoryInductor, ValueType: domain.ValueTypeNumber, DisplayName: "DC Resistance", Unit: strptr("ohm")},
	{Key: AttrPackage, Category: domain.CategoryInductor, ValueType: domain.ValueTypeText, DisplayName: "Package"},
	{Key: AttrInductorType, Category: domain.CategoryInductor, ValueType: domain.ValueTypeText, DisplayName: "Inductor Type"},
}
