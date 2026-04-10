package domain_test

import (
	"testing"

	"trace/internal/domain"
	"trace/internal/domain/registry"
)

func strp(s string) *string { return &s }

func TestGetAttribute_Found(t *testing.T) {
	v := 10000.0
	c := domain.Component{
		Attributes: []domain.AttributeValue{
			{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Number: &v, Unit: "ohm"},
			{Key: registry.AttrPackage, ValueType: domain.ValueTypeText, Text: strp("0402")},
		},
	}
	attr, ok := c.GetAttribute(registry.AttrResistanceOhms)
	if !ok {
		t.Fatal("expected to find attribute")
	}
	if attr.Number == nil || *attr.Number != 10000.0 {
		t.Errorf("unexpected value: %v", attr.Number)
	}
}

func TestGetAttribute_NotFound(t *testing.T) {
	c := domain.Component{}
	_, ok := c.GetAttribute("nonexistent")
	if ok {
		t.Fatal("expected not found")
	}
}

func TestHasAttribute(t *testing.T) {
	c := domain.Component{
		Attributes: []domain.AttributeValue{
			{Key: registry.AttrPackage, ValueType: domain.ValueTypeText, Text: strp("0402")},
		},
	}
	if !c.HasAttribute(registry.AttrPackage) {
		t.Error("expected HasAttribute to return true for package")
	}
	if c.HasAttribute(registry.AttrResistanceOhms) {
		t.Error("expected HasAttribute to return false for missing key")
	}
}

func TestAttributeIndex_ReturnsMap(t *testing.T) {
	v := 10000.0
	c := domain.Component{
		Attributes: []domain.AttributeValue{
			{Key: registry.AttrResistanceOhms, ValueType: domain.ValueTypeNumber, Number: &v, Unit: "ohm"},
			{Key: registry.AttrPackage, ValueType: domain.ValueTypeText, Text: strp("0402")},
		},
	}
	idx := c.AttributeIndex()
	if len(idx) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(idx))
	}
	attr, ok := idx[registry.AttrResistanceOhms]
	if !ok {
		t.Fatal("expected resistance key in index")
	}
	if attr.Number == nil || *attr.Number != 10000.0 {
		t.Errorf("unexpected value: %v", attr.Number)
	}
}

func TestAttributeIndex_EmptyComponent(t *testing.T) {
	idx := domain.Component{}.AttributeIndex()
	if len(idx) != 0 {
		t.Errorf("expected empty index, got %d entries", len(idx))
	}
}
