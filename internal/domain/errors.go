package domain

import "fmt"

type ErrUnknownAttribute struct {
	Key      string
	Category Category
}

func (e ErrUnknownAttribute) Error() string {
	return fmt.Sprintf("attribute %q is not defined for category %q", e.Key, e.Category)
}

type ErrAttributeTypeMismatch struct {
	Key  string
	Want ValueType
	Got  ValueType
}

func (e ErrAttributeTypeMismatch) Error() string {
	return fmt.Sprintf("attribute %q: expected value type %q, got %q", e.Key, e.Want, e.Got)
}

type ErrAttributeUnitMismatch struct {
	Key  string
	Want string
	Got  string
}

func (e ErrAttributeUnitMismatch) Error() string {
	return fmt.Sprintf("attribute %q: expected unit %q, got %q", e.Key, e.Want, e.Got)
}

type ErrUnknownConstraint struct {
	Key      string
	Category Category
}

func (e ErrUnknownConstraint) Error() string {
	return fmt.Sprintf("constraint key %q is not defined for category %q", e.Key, e.Category)
}

type ErrConstraintTypeMismatch struct {
	Key  string
	Want ValueType
	Got  ValueType
}

func (e ErrConstraintTypeMismatch) Error() string {
	return fmt.Sprintf("constraint %q: expected value type %q, got %q", e.Key, e.Want, e.Got)
}

type ErrConstraintUnitMismatch struct {
	Key  string
	Want string
	Got  string
}

func (e ErrConstraintUnitMismatch) Error() string {
	return fmt.Sprintf("constraint %q: expected unit %q, got %q", e.Key, e.Want, e.Got)
}

type ErrInvalidOperator struct {
	Key      string
	Operator Operator
}

func (e ErrInvalidOperator) Error() string {
	return fmt.Sprintf("constraint %q: operator %q is not valid for its value type", e.Key, e.Operator)
}

type ErrCategoryMismatch struct {
	RequirementCategory Category
	ComponentCategory   Category
}

func (e ErrCategoryMismatch) Error() string {
	return fmt.Sprintf("component category %q does not match requirement category %q", e.ComponentCategory, e.RequirementCategory)
}

type ErrRequirementNotSatisfied struct {
	ComponentID   string
	RequirementID string
}

func (e ErrRequirementNotSatisfied) Error() string {
	return fmt.Sprintf("component %q does not satisfy requirement %q constraints", e.ComponentID, e.RequirementID)
}

type ErrNotFound struct {
	ID string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("record %q not found", e.ID)
}
