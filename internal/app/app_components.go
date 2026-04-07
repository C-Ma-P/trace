package app

import (
	"context"
	"time"

	"componentmanager/internal/domain"
)

func (a *App) ListComponents(filter ComponentFilterInput) ([]ComponentResponse, error) {
	if err := a.checkReady(); err != nil {
		return nil, err
	}
	f := domain.ComponentFilter{
		Manufacturer: filter.Manufacturer,
		MPN:          filter.MPN,
		Package:      filter.Package,
		Text:         filter.Text,
	}
	if filter.Category != "" {
		cat := domain.Category(filter.Category)
		f.Category = &cat
	}
	components, err := a.svc.FindComponents(context.Background(), f)
	if err != nil {
		return nil, err
	}
	out := make([]ComponentResponse, len(components))
	for i, c := range components {
		out[i] = componentToResponse(c)
	}
	if a.bagRepo != nil && len(out) > 0 {
		ids := make([]string, len(out))
		for i, c := range out {
			ids[i] = c.ID
		}
		imageURLs := a.bagRepo.FindComponentImageURLs(context.Background(), ids)
		for i := range out {
			out[i].ImageURL = imageURLs[out[i].ID]
		}
	}
	return out, nil
}

func (a *App) GetComponent(id string) (ComponentResponse, error) {
	if err := a.checkReady(); err != nil {
		return ComponentResponse{}, err
	}
	c, err := a.svc.GetComponent(context.Background(), id)
	if err != nil {
		return ComponentResponse{}, err
	}
	return componentToResponse(c), nil
}

func (a *App) CreateComponent(req CreateComponentInput) (ComponentResponse, error) {
	if err := a.checkReady(); err != nil {
		return ComponentResponse{}, err
	}
	c, err := a.svc.CreateComponent(context.Background(), domain.Component{
		Category:     domain.Category(req.Category),
		MPN:          req.MPN,
		Manufacturer: req.Manufacturer,
		Package:      req.Package,
		Description:  req.Description,
	})
	if err != nil {
		return ComponentResponse{}, err
	}
	return componentToResponse(c), nil
}

func (a *App) UpdateComponentMetadata(req UpdateMetadataInput) (ComponentResponse, error) {
	if err := a.checkReady(); err != nil {
		return ComponentResponse{}, err
	}
	c, err := a.svc.UpdateComponentMetadata(context.Background(), domain.Component{
		ID:           req.ID,
		MPN:          req.MPN,
		Manufacturer: req.Manufacturer,
		Package:      req.Package,
		Description:  req.Description,
	})
	if err != nil {
		return ComponentResponse{}, err
	}
	return componentToResponse(c), nil
}

func (a *App) UpdateComponentInventory(req UpdateInventoryInput) (ComponentResponse, error) {
	if err := a.checkReady(); err != nil {
		return ComponentResponse{}, err
	}
	c, err := a.svc.UpdateComponentInventory(context.Background(), domain.Component{
		ID:           req.ID,
		Quantity:     req.Quantity,
		QuantityMode: domain.QuantityMode(req.QuantityMode),
		Location:     req.Location,
	})
	if err != nil {
		return ComponentResponse{}, err
	}
	return componentToResponse(c), nil
}

func (a *App) AdjustComponentQuantity(id string, delta int) (ComponentResponse, error) {
	if err := a.checkReady(); err != nil {
		return ComponentResponse{}, err
	}
	c, err := a.svc.AdjustComponentQuantity(context.Background(), id, delta)
	if err != nil {
		return ComponentResponse{}, err
	}
	return componentToResponse(c), nil
}

func (a *App) DeleteComponent(id string) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	return a.svc.DeleteComponent(context.Background(), id)
}

func (a *App) ReplaceComponentAttributes(componentID string, attrs []AttributeValueInput) error {
	if err := a.checkReady(); err != nil {
		return err
	}
	domainAttrs := make([]domain.AttributeValue, len(attrs))
	for i, a := range attrs {
		domainAttrs[i] = domain.AttributeValue{
			Key:       a.Key,
			ValueType: domain.ValueType(a.ValueType),
			Text:      a.Text,
			Number:    a.Number,
			Bool:      a.Bool,
			Unit:      a.Unit,
		}
	}
	return a.svc.ReplaceComponentAttributes(context.Background(), componentID, domainAttrs)
}

func componentToResponse(c domain.Component) ComponentResponse {
	attrs := make([]AttributeValueResponse, len(c.Attributes))
	for i, a := range c.Attributes {
		attrs[i] = AttributeValueResponse{
			Key:       a.Key,
			ValueType: string(a.ValueType),
			Text:      a.Text,
			Number:    a.Number,
			Bool:      a.Bool,
			Unit:      a.Unit,
		}
	}
	mode := string(c.QuantityMode)
	if mode == "" {
		mode = "unknown"
	}
	return ComponentResponse{
		ID:           c.ID,
		Category:     string(c.Category),
		MPN:          c.MPN,
		Manufacturer: c.Manufacturer,
		Package:      c.Package,
		Description:  c.Description,
		Quantity:     c.Quantity,
		QuantityMode: mode,
		Location:     c.Location,
		Attributes:   attrs,
		CreatedAt:    c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    c.UpdatedAt.Format(time.RFC3339),
	}
}
