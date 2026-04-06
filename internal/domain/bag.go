package domain

import "time"

// InventoryBag maps a QR code on a physical bag/container to a component.
type InventoryBag struct {
	ID          string    `db:"id"`
	Label       string    `db:"label"`
	QRData      string    `db:"qr_data"`
	ComponentID string    `db:"component_id"`
	CreatedAt   time.Time `db:"created_at"`
	UpdatedAt   time.Time `db:"updated_at"`
}
