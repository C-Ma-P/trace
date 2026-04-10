package registry_test

import (
	"errors"
	"testing"

	"trace/internal/domain"
	"trace/internal/domain/registry"
)

func numConstraint(key string, op domain.Operator, v float64, unit string) domain.RequirementConstraint {
	return domain.RequirementConstraint{Key: key, ValueType: domain.ValueTypeNumber, Operator: op, Number: &v, Unit: unit}
}

func textConstraint(key string, v string) domain.RequirementConstraint {
	return domain.RequirementConstraint{Key: key, ValueType: domain.ValueTypeText, Operator: domain.OperatorEqual, Text: &v}
}

func TestValidateConstraints_ResistorValid(t *testing.T) {
	constraints := []domain.RequirementConstraint{
		numConstraint(registry.AttrResistanceOhms, domain.OperatorEqual, 10000, "ohm"),
		numConstraint(registry.AttrTolerancePercent, domain.OperatorLTE, 1, "percent"),
		numConstraint(registry.AttrPowerW, domain.OperatorGTE, 0.125, "W"),
		textConstraint(registry.AttrPackage, "0402"),
	}
	if err := registry.ValidateConstraints(domain.CategoryResistor, constraints); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateConstraints_CapacitorValid(t *testing.T) {
	constraints := []domain.RequirementConstraint{
		numConstraint(registry.AttrCapacitanceF, domain.OperatorGTE, 10e-9, "F"),
		numConstraint(registry.AttrVoltageV, domain.OperatorGTE, 10, "V"),
		textConstraint(registry.AttrDielectric, "X7R"),
	}
	if err := registry.ValidateConstraints(domain.CategoryCapacitor, constraints); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateConstraints_UnknownKey(t *testing.T) {
	constraints := []domain.RequirementConstraint{
		numConstraint("unknown_key", domain.OperatorEqual, 1, ""),
	}
	err := registry.ValidateConstraints(domain.CategoryResistor, constraints)
	var target domain.ErrUnknownConstraint
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownConstraint, got %v", err)
	}
	if target.Key != "unknown_key" {
		t.Errorf("wrong key: %q", target.Key)
	}
}

func TestValidateConstraints_CapacitorKeyOnResistor(t *testing.T) {
	constraints := []domain.RequirementConstraint{
		numConstraint(registry.AttrCapacitanceF, domain.OperatorEqual, 100e-9, "F"),
	}
	err := registry.ValidateConstraints(domain.CategoryResistor, constraints)
	var target domain.ErrUnknownConstraint
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownConstraint, got %v", err)
	}
}

func TestValidateConstraints_WrongValueType(t *testing.T) {
	v := "10k"
	constraints := []domain.RequirementConstraint{
		{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeText, Operator: domain.OperatorEqual, Text: &v, Unit: "ohm"},
	}
	err := registry.ValidateConstraints(domain.CategoryResistor, constraints)
	var target domain.ErrConstraintTypeMismatch
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrConstraintTypeMismatch, got %v", err)
	}
	if target.Want != domain.ValueTypeNumber || target.Got != domain.ValueTypeText {
		t.Errorf("wrong mismatch: want=%q got=%q", target.Want, target.Got)
	}
}

func TestValidateConstraints_WrongUnit(t *testing.T) {
	constraints := []domain.RequirementConstraint{
		numConstraint(registry.AttrResistanceOhms, domain.OperatorEqual, 10000, "kilohm"),
	}
	err := registry.ValidateConstraints(domain.CategoryResistor, constraints)
	var target domain.ErrConstraintUnitMismatch
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrConstraintUnitMismatch, got %v", err)
	}
	if target.Want != "ohm" || target.Got != "kilohm" {
		t.Errorf("wrong unit: want=%q got=%q", target.Want, target.Got)
	}
}

func TestValidateConstraints_InvalidOperatorForText(t *testing.T) {
	v := "0402"
	constraints := []domain.RequirementConstraint{
		{Key: registry.AttrPackage, ValueType: domain.ValueTypeText, Operator: domain.OperatorGTE, Text: &v},
	}
	err := registry.ValidateConstraints(domain.CategoryResistor, constraints)
	var target domain.ErrInvalidOperator
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrInvalidOperator, got %v", err)
	}
	if target.Operator != domain.OperatorGTE {
		t.Errorf("wrong operator: %q", target.Operator)
	}
}

func TestValidateConstraints_InvalidOperatorForNumber(t *testing.T) {
	n := 1000.0
	constraints := []domain.RequirementConstraint{
		{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Operator: "between", Number: &n, Unit: "ohm"},
	}
	err := registry.ValidateConstraints(domain.CategoryResistor, constraints)
	var target domain.ErrInvalidOperator
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrInvalidOperator, got %v", err)
	}
}

func TestValidateConstraints_EmptyIsValid(t *testing.T) {
	if err := registry.ValidateConstraints(domain.CategoryResistor, nil); err != nil {
		t.Fatalf("empty constraints should be valid, got: %v", err)
	}
}

func TestValidateConstraints_InductorValid(t *testing.T) {
	constraints := []domain.RequirementConstraint{
		numConstraint(registry.AttrInductanceH, domain.OperatorGTE, 10e-6, "H"),
		numConstraint(registry.AttrCurrentA, domain.OperatorGTE, 1.0, "A"),
		numConstraint(registry.AttrDCROhms, domain.OperatorLTE, 0.1, "ohm"),
		textConstraint(registry.AttrPackage, "0805"),
	}
	if err := registry.ValidateConstraints(domain.CategoryInductor, constraints); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateConstraints_InductorKeyOnResistor(t *testing.T) {
	constraints := []domain.RequirementConstraint{
		numConstraint(registry.AttrInductanceH, domain.OperatorEqual, 10e-6, "H"),
	}
	err := registry.ValidateConstraints(domain.CategoryResistor, constraints)
	var target domain.ErrUnknownConstraint
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownConstraint, got %v", err)
	}
}

func TestValidateConstraints_GTEAndLTEAreValid(t *testing.T) {
	n := 10000.0
	gte := domain.RequirementConstraint{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Operator: domain.OperatorGTE, Number: &n, Unit: "ohm"}
	lte := domain.RequirementConstraint{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Operator: domain.OperatorLTE, Number: &n, Unit: "ohm"}

	if err := registry.ValidateConstraints(domain.CategoryResistor, []domain.RequirementConstraint{gte}); err != nil {
		t.Errorf("gte should be valid: %v", err)
	}
	if err := registry.ValidateConstraints(domain.CategoryResistor, []domain.RequirementConstraint{lte}); err != nil {
		t.Errorf("lte should be valid: %v", err)
	}
}
