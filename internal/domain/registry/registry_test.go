package registry_test

import (
	"errors"
	"testing"

	"trace/internal/domain"
	"trace/internal/domain/registry"
)

func numAttr(key string, v float64, unit string) domain.AttributeValue {
	return domain.AttributeValue{Key: key, ValueType: domain.ValueTypeNumber, Number: &v, Unit: unit}
}

func textAttr(key, v string) domain.AttributeValue {
	return domain.AttributeValue{Key: key, ValueType: domain.ValueTypeText, Text: &v}
}

func TestValidateAttributes_ResistorValid(t *testing.T) {
	attrs := []domain.AttributeValue{
		numAttr(registry.AttrResistanceOhms, 10000, "ohm"),
		numAttr(registry.AttrTolerancePercent, 1, "percent"),
		numAttr(registry.AttrPowerW, 0.25, "W"),
		textAttr(registry.AttrPackage, "0402"),
		textAttr(registry.AttrResistorType, "thick film"),
	}
	if err := registry.ValidateAttributes(domain.CategoryResistor, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_CapacitorValid(t *testing.T) {
	attrs := []domain.AttributeValue{
		numAttr(registry.AttrCapacitanceF, 100e-9, "F"),
		numAttr(registry.AttrTolerancePercent, 10, "percent"),
		numAttr(registry.AttrVoltageV, 50, "V"),
		textAttr(registry.AttrPackage, "0603"),
		textAttr(registry.AttrDielectric, "X7R"),
		textAttr(registry.AttrCapacitorType, "ceramic"),
	}
	if err := registry.ValidateAttributes(domain.CategoryCapacitor, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateAttributes_UnknownKey(t *testing.T) {
	attrs := []domain.AttributeValue{
		numAttr("unknown_field", 1, ""),
	}
	err := registry.ValidateAttributes(domain.CategoryResistor, attrs)
	var target domain.ErrUnknownAttribute
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownAttribute, got %v", err)
	}
	if target.Key != "unknown_field" {
		t.Errorf("wrong key in error: %q", target.Key)
	}
}

func TestValidateAttributes_CapacitorKeyOnResistor(t *testing.T) {
	attrs := []domain.AttributeValue{
		numAttr(registry.AttrCapacitanceF, 100e-9, "F"),
	}
	err := registry.ValidateAttributes(domain.CategoryResistor, attrs)
	var target domain.ErrUnknownAttribute
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrUnknownAttribute, got %v", err)
	}
}

func TestValidateAttributes_WrongValueType(t *testing.T) {
	v := "10k"
	attrs := []domain.AttributeValue{
		{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeText, Text: &v, Unit: "ohm"},
	}
	err := registry.ValidateAttributes(domain.CategoryResistor, attrs)
	var target domain.ErrAttributeTypeMismatch
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrAttributeTypeMismatch, got %v", err)
	}
	if target.Want != domain.ValueTypeNumber || target.Got != domain.ValueTypeText {
		t.Errorf("wrong mismatch detail: want=%q got=%q", target.Want, target.Got)
	}
}

func TestValidateAttributes_WrongUnit(t *testing.T) {
	attrs := []domain.AttributeValue{
		numAttr(registry.AttrResistanceOhms, 10000, "kilo-ohm"),
	}
	err := registry.ValidateAttributes(domain.CategoryResistor, attrs)
	var target domain.ErrAttributeUnitMismatch
	if !errors.As(err, &target) {
		t.Fatalf("expected ErrAttributeUnitMismatch, got %v", err)
	}
	if target.Want != "ohm" || target.Got != "kilo-ohm" {
		t.Errorf("wrong unit detail: want=%q got=%q", target.Want, target.Got)
	}
}

func TestValidateAttributes_EmptyIsValid(t *testing.T) {
	if err := registry.ValidateAttributes(domain.CategoryResistor, nil); err != nil {
		t.Fatalf("empty attrs should be valid, got: %v", err)
	}
}

func TestValidateAttributes_InductorValid(t *testing.T) {
	attrs := []domain.AttributeValue{
		numAttr(registry.AttrInductanceH, 10e-6, "H"),
		numAttr(registry.AttrTolerancePercent, 20, "percent"),
		numAttr(registry.AttrCurrentA, 1.5, "A"),
		numAttr(registry.AttrDCROhms, 0.05, "ohm"),
		textAttr(registry.AttrPackage, "0805"),
		textAttr(registry.AttrInductorType, "ferrite"),
	}
	if err := registry.ValidateAttributes(domain.CategoryInductor, attrs); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCategories_ContainsAllThree(t *testing.T) {
	cats := registry.Categories()
	found := make(map[domain.Category]bool, len(cats))
	for _, c := range cats {
		found[c] = true
	}
	for _, want := range []domain.Category{domain.CategoryResistor, domain.CategoryCapacitor, domain.CategoryInductor} {
		if !found[want] {
			t.Errorf("Categories() missing %q", want)
		}
	}
}
