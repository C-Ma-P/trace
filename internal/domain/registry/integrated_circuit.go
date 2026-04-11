package registry

import "github.com/C-Ma-P/trace/internal/domain"

var integratedCircuitRequirementDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryIntegratedCircuit, ValueType: domain.ValueTypeText, DisplayName: "Package"},
}
