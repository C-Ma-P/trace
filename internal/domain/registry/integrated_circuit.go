package registry

import "componentmanager/internal/domain"

var integratedCircuitRequirementDefs = []domain.AttributeDefinition{
	{Key: AttrPackage, Category: domain.CategoryIntegratedCircuit, ValueType: domain.ValueTypeText, DisplayName: "Package"},
}
