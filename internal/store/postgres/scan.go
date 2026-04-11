package postgres

import "github.com/C-Ma-P/trace/internal/domain"

// attributeRow is a flat scan target for component_attributes rows.
// It includes component_id for grouping results by component.
type attributeRow struct {
	ComponentID string           `db:"component_id"`
	Key         string           `db:"key"`
	ValueType   domain.ValueType `db:"value_type"`
	Text        *string          `db:"text_value"`
	Number      *float64         `db:"number_value"`
	Bool        *bool            `db:"bool_value"`
	Unit        string           `db:"unit"`
}

func (r attributeRow) toAttributeValue() domain.AttributeValue {
	return domain.AttributeValue{
		Key:       r.Key,
		ValueType: r.ValueType,
		Text:      r.Text,
		Number:    r.Number,
		Bool:      r.Bool,
		Unit:      r.Unit,
	}
}

// constraintRow is a flat scan target for requirement_constraints rows.
// It includes requirement_id for grouping results by requirement.
type constraintRow struct {
	RequirementID string           `db:"requirement_id"`
	Key           string           `db:"key"`
	ValueType     domain.ValueType `db:"value_type"`
	Operator      domain.Operator  `db:"operator"`
	Text          *string          `db:"text_value"`
	Number        *float64         `db:"number_value"`
	Bool          *bool            `db:"bool_value"`
	Unit          string           `db:"unit"`
}

func (r constraintRow) toRequirementConstraint() domain.RequirementConstraint {
	return domain.RequirementConstraint{
		Key:       r.Key,
		ValueType: r.ValueType,
		Operator:  r.Operator,
		Text:      r.Text,
		Number:    r.Number,
		Bool:      r.Bool,
		Unit:      r.Unit,
	}
}
