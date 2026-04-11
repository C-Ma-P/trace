package service

import (
	"context"
	"fmt"

	"github.com/C-Ma-P/trace/internal/domain"
	"github.com/C-Ma-P/trace/internal/domain/registry"
)

func (s *Service) CreateComponent(ctx context.Context, component domain.Component) (domain.Component, error) {
	if component.ID == "" {
		component.ID = newID()
	}

	if err := registry.ValidateAttributes(component.Category, component.Attributes); err != nil {
		return domain.Component{}, err
	}

	return s.components.CreateComponent(ctx, component)
}

func (s *Service) GetComponent(ctx context.Context, id string) (domain.Component, error) {
	return s.components.GetComponent(ctx, id)
}

func (s *Service) UpdateComponentMetadata(ctx context.Context, component domain.Component) (domain.Component, error) {
	return s.components.UpdateComponentMetadata(ctx, component)
}

func (s *Service) ReplaceComponentAttributes(ctx context.Context, componentID string, attrs []domain.AttributeValue) error {
	c, err := s.components.GetComponent(ctx, componentID)
	if err != nil {
		return err
	}
	if err := registry.ValidateAttributes(c.Category, attrs); err != nil {
		return err
	}
	return s.components.ReplaceComponentAttributes(ctx, componentID, attrs)
}

func (s *Service) FindComponents(ctx context.Context, filter domain.ComponentFilter) ([]domain.Component, error) {
	return s.components.FindComponents(ctx, filter)
}

func (s *Service) DeleteComponent(ctx context.Context, id string) error {
	return s.components.DeleteComponent(ctx, id)
}

func (s *Service) UpdateComponentInventory(ctx context.Context, component domain.Component) (domain.Component, error) {
	if component.QuantityMode == "" {
		component.QuantityMode = domain.QuantityModeUnknown
	}
	if err := component.ValidateInventory(); err != nil {
		return domain.Component{}, err
	}
	return s.components.UpdateComponentInventory(ctx, component)
}

// AdjustComponentQuantity increments or decrements the component's quantity by delta.
// Only valid when quantity_mode is exact or approximate.
func (s *Service) AdjustComponentQuantity(ctx context.Context, id string, delta int) (domain.Component, error) {
	c, err := s.components.GetComponent(ctx, id)
	if err != nil {
		return domain.Component{}, err
	}
	if c.QuantityMode == domain.QuantityModeUnknown || c.QuantityMode == "" {
		return domain.Component{}, fmt.Errorf("cannot adjust quantity when quantity_mode is unknown")
	}
	current := 0
	if c.Quantity != nil {
		current = *c.Quantity
	}
	next := current + delta
	if next < 0 {
		next = 0
	}
	c.Quantity = &next
	return s.components.UpdateComponentInventory(ctx, c)
}

func (s *Service) StampInventory(ctx context.Context, id string, qty int) (domain.Component, error) {
	c, err := s.components.GetComponent(ctx, id)
	if err != nil {
		return domain.Component{}, err
	}
	if c.QuantityMode == domain.QuantityModeUnknown || c.QuantityMode == "" {
		c.QuantityMode = domain.QuantityModeExact
		c.Quantity = &qty
		return s.components.UpdateComponentInventory(ctx, c)
	}
	current := 0
	if c.Quantity != nil {
		current = *c.Quantity
	}
	next := current + qty
	if next < 0 {
		next = 0
	}
	c.Quantity = &next
	return s.components.UpdateComponentInventory(ctx, c)
}
