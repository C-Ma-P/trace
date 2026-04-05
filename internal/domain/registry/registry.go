package registry

import "componentmanager/internal/domain"

var canonical = map[domain.Category]map[string]domain.AttributeDefinition{
	domain.CategoryResistor:  indexByKey(resistorDefs),
	domain.CategoryCapacitor: indexByKey(capacitorDefs),
	domain.CategoryInductor:  indexByKey(inductorDefs),
}

var requirementCanonical = map[domain.Category]map[string]domain.AttributeDefinition{
	domain.CategoryResistor:          indexByKey(append(sharedRequirementDefs(domain.CategoryResistor), resistorDefs...)),
	domain.CategoryCapacitor:         indexByKey(append(sharedRequirementDefs(domain.CategoryCapacitor), capacitorDefs...)),
	domain.CategoryInductor:          indexByKey(append(sharedRequirementDefs(domain.CategoryInductor), inductorDefs...)),
	domain.CategoryIntegratedCircuit: indexByKey(append(sharedRequirementDefs(domain.CategoryIntegratedCircuit), integratedCircuitRequirementDefs...)),
}

func Categories() []domain.Category {
	categories := make([]domain.Category, 0, len(canonical))
	for cat := range canonical {
		categories = append(categories, cat)
	}
	return categories
}

func DefinitionsForCategory(category domain.Category) []domain.AttributeDefinition {
	m, ok := canonical[category]
	if !ok {
		return nil
	}
	out := make([]domain.AttributeDefinition, 0, len(m))
	for _, def := range m {
		out = append(out, def)
	}
	return out
}

func LookupDefinition(category domain.Category, key string) (domain.AttributeDefinition, bool) {
	m, ok := canonical[category]
	if !ok {
		return domain.AttributeDefinition{}, false
	}
	def, ok := m[key]
	return def, ok
}

func ConstraintDefinitionsForCategory(category domain.Category) []domain.AttributeDefinition {
	m, ok := requirementCanonical[category]
	if !ok {
		return nil
	}
	out := make([]domain.AttributeDefinition, 0, len(m))
	for _, def := range m {
		out = append(out, def)
	}
	return out
}

func LookupConstraintDefinition(category domain.Category, key string) (domain.AttributeDefinition, bool) {
	m, ok := requirementCanonical[category]
	if !ok {
		return domain.AttributeDefinition{}, false
	}
	def, ok := m[key]
	return def, ok
}

func ValidateAttributes(category domain.Category, attrs []domain.AttributeValue) error {
	for _, attr := range attrs {
		def, ok := LookupDefinition(category, attr.Key)
		if !ok {
			return domain.ErrUnknownAttribute{Key: attr.Key, Category: category}
		}
		if attr.ValueType != def.ValueType {
			return domain.ErrAttributeTypeMismatch{Key: attr.Key, Want: def.ValueType, Got: attr.ValueType}
		}
		if def.Unit != nil && attr.Unit != *def.Unit {
			return domain.ErrAttributeUnitMismatch{Key: attr.Key, Want: *def.Unit, Got: attr.Unit}
		}
	}
	return nil
}

func ValidateConstraints(category domain.Category, constraints []domain.RequirementConstraint) error {
	for _, c := range constraints {
		def, ok := LookupConstraintDefinition(category, c.Key)
		if !ok {
			return domain.ErrUnknownConstraint{Key: c.Key, Category: category}
		}
		if c.ValueType != def.ValueType {
			return domain.ErrConstraintTypeMismatch{Key: c.Key, Want: def.ValueType, Got: c.ValueType}
		}
		if def.Unit != nil && c.Unit != *def.Unit {
			return domain.ErrConstraintUnitMismatch{Key: c.Key, Want: *def.Unit, Got: c.Unit}
		}
		if err := validOperatorForType(c.Key, c.ValueType, c.Operator); err != nil {
			return err
		}
	}
	return nil
}

func sharedRequirementDefs(category domain.Category) []domain.AttributeDefinition {
	return []domain.AttributeDefinition{
		{Key: AttrManufacturer, Category: category, ValueType: domain.ValueTypeText, DisplayName: "Manufacturer"},
		{Key: AttrMPN, Category: category, ValueType: domain.ValueTypeText, DisplayName: "MPN"},
	}
}

func validOperatorForType(key string, vt domain.ValueType, op domain.Operator) error {
	switch vt {
	case domain.ValueTypeText, domain.ValueTypeBool:
		if op != domain.OperatorEqual {
			return domain.ErrInvalidOperator{Key: key, Operator: op}
		}
	case domain.ValueTypeNumber:
		switch op {
		case domain.OperatorEqual, domain.OperatorGTE, domain.OperatorLTE:
		default:
			return domain.ErrInvalidOperator{Key: key, Operator: op}
		}
	}
	return nil
}

func indexByKey(defs []domain.AttributeDefinition) map[string]domain.AttributeDefinition {
	m := make(map[string]domain.AttributeDefinition, len(defs))
	for _, d := range defs {
		m[d.Key] = d
	}
	return m
}

func strptr(s string) *string { return &s }
