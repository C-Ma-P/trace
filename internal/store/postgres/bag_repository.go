package postgres

import (
	"context"
	"database/sql"
	"errors"

	"componentmanager/internal/domain"
)

type BagRepository struct {
	store *Store
}

func NewBagRepository(store *Store) *BagRepository {
	return &BagRepository{store: store}
}

func (r *BagRepository) CreateBag(ctx context.Context, bag domain.InventoryBag) (domain.InventoryBag, error) {
	err := r.store.db.QueryRowxContext(ctx, `
INSERT INTO inventory_bags (id, label, qr_data, component_id)
VALUES ($1, $2, $3, $4)
RETURNING created_at, updated_at
`, bag.ID, bag.Label, bag.QRData, bag.ComponentID,
	).Scan(&bag.CreatedAt, &bag.UpdatedAt)
	if err != nil {
		return domain.InventoryBag{}, err
	}
	return bag, nil
}

func (r *BagRepository) GetBagByQRData(ctx context.Context, qrData string) (domain.InventoryBag, error) {
	var bag domain.InventoryBag
	err := r.store.db.GetContext(ctx, &bag, `
SELECT id, label, qr_data, component_id, created_at, updated_at
FROM inventory_bags
WHERE qr_data = $1
`, qrData)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.InventoryBag{}, domain.ErrNotFound{ID: qrData}
		}
		return domain.InventoryBag{}, err
	}
	return bag, nil
}

func (r *BagRepository) ListBagsByComponent(ctx context.Context, componentID string) ([]domain.InventoryBag, error) {
	var bags []domain.InventoryBag
	err := r.store.db.SelectContext(ctx, &bags, `
SELECT id, label, qr_data, component_id, created_at, updated_at
FROM inventory_bags
WHERE component_id = $1
ORDER BY created_at
`, componentID)
	if err != nil {
		return nil, err
	}
	return bags, nil
}

func (r *BagRepository) DeleteBag(ctx context.Context, id string) error {
	_, err := r.store.db.ExecContext(ctx, `DELETE FROM inventory_bags WHERE id = $1`, id)
	return err
}

// FindComponentImageURL returns the first non-empty image_url from saved
// supplier offers linked to the given component. Returns empty string if none.
func (r *BagRepository) FindComponentImageURL(ctx context.Context, componentID string) string {
	var url string
	err := r.store.db.GetContext(ctx, &url, `
SELECT image_url FROM saved_supplier_offers
WHERE linked_component_id = $1 AND image_url != ''
LIMIT 1
`, componentID)
	if err != nil {
		return ""
	}
	return url
}
