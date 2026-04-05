package domain

import (
	"fmt"
	"time"
)

// QuantityMode describes how the component's quantity should be interpreted.
type QuantityMode string

const (
	QuantityModeExact       QuantityMode = "exact"
	QuantityModeApproximate QuantityMode = "approximate"
	QuantityModeUnknown     QuantityMode = "unknown"
)

type Component struct {
	ID           string           `db:"id"`
	Category     Category         `db:"category"`
	MPN          string           `db:"mpn"`
	Manufacturer string           `db:"manufacturer"`
	Package      string           `db:"package"`
	Description  string           `db:"description"`
	Quantity     *int             `db:"quantity"`
	QuantityMode QuantityMode     `db:"quantity_mode"`
	Location     string           `db:"location"`
	Attributes   []AttributeValue `db:"-"`
	CreatedAt    time.Time        `db:"created_at"`
	UpdatedAt    time.Time        `db:"updated_at"`

	SelectedSymbolAssetID    *string `db:"selected_symbol_asset_id"`
	SelectedFootprintAssetID *string `db:"selected_footprint_asset_id"`
	Selected3DModelAssetID   *string `db:"selected_3d_model_asset_id"`
	SelectedDatasheetAssetID *string `db:"selected_datasheet_asset_id"`
}

// ValidateInventory checks that inventory field combinations are sensible.
func (c Component) ValidateInventory() error {
	switch c.QuantityMode {
	case QuantityModeExact, QuantityModeApproximate:
		if c.Quantity == nil {
			return fmt.Errorf("quantity_mode %q requires a non-nil quantity", c.QuantityMode)
		}
		if *c.Quantity < 0 {
			return fmt.Errorf("quantity must be >= 0")
		}
	case QuantityModeUnknown, "":
		// quantity may be nil; that is expected
	default:
		return fmt.Errorf("unknown quantity_mode %q", c.QuantityMode)
	}
	return nil
}

func (c Component) GetAttribute(key string) (AttributeValue, bool) {
	for _, a := range c.Attributes {
		if a.Key == key {
			return a, true
		}
	}
	return AttributeValue{}, false
}

func (c Component) HasAttribute(key string) bool {
	_, ok := c.GetAttribute(key)
	return ok
}

func (c Component) AttributeIndex() map[string]AttributeValue {
	idx := make(map[string]AttributeValue, len(c.Attributes))
	for _, a := range c.Attributes {
		idx[a.Key] = a
	}
	return idx
}

type ComponentMatch struct {
	Component      Component
	OnHandQuantity int
	Score          int
}
