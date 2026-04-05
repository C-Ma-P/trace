package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"componentmanager/internal/domain"
)

type ComponentAssetRepository struct {
	store *Store
}

func NewComponentAssetRepository(store *Store) *ComponentAssetRepository {
	return &ComponentAssetRepository{store: store}
}

func (r *ComponentAssetRepository) CreateComponentAsset(ctx context.Context, asset domain.ComponentAsset) (domain.ComponentAsset, error) {
	if err := r.store.db.QueryRowxContext(ctx, `
		insert into component_assets(id, component_id, asset_type, source, status, label, url_or_path, preview_url, metadata_json)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		returning created_at, updated_at
	`, asset.ID, asset.ComponentID, asset.AssetType, asset.Source, asset.Status,
		asset.Label, asset.URLOrPath, asset.PreviewURL, asset.MetadataJSON,
	).Scan(&asset.CreatedAt, &asset.UpdatedAt); err != nil {
		return domain.ComponentAsset{}, err
	}
	return asset, nil
}

func (r *ComponentAssetRepository) GetComponentAsset(ctx context.Context, id string) (domain.ComponentAsset, error) {
	var asset domain.ComponentAsset
	if err := r.store.db.GetContext(ctx, &asset, `
		select id, component_id, asset_type, source, status, label, url_or_path,
		       preview_url, metadata_json, created_at, updated_at
		from component_assets
		where id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ComponentAsset{}, domain.ErrNotFound{ID: id}
		}
		return domain.ComponentAsset{}, err
	}
	return asset, nil
}

func (r *ComponentAssetRepository) ListComponentAssets(ctx context.Context, componentID string) ([]domain.ComponentAsset, error) {
	var assets []domain.ComponentAsset
	if err := r.store.db.SelectContext(ctx, &assets, `
		select id, component_id, asset_type, source, status, label, url_or_path,
		       preview_url, metadata_json, created_at, updated_at
		from component_assets
		where component_id = $1
		order by asset_type, created_at
	`, componentID); err != nil {
		return nil, err
	}
	return assets, nil
}

func (r *ComponentAssetRepository) ListComponentAssetsByType(ctx context.Context, componentID string, assetType domain.AssetType) ([]domain.ComponentAsset, error) {
	var assets []domain.ComponentAsset
	if err := r.store.db.SelectContext(ctx, &assets, `
		select id, component_id, asset_type, source, status, label, url_or_path,
		       preview_url, metadata_json, created_at, updated_at
		from component_assets
		where component_id = $1 and asset_type = $2
		order by created_at
	`, componentID, assetType); err != nil {
		return nil, err
	}
	return assets, nil
}

func (r *ComponentAssetRepository) UpdateComponentAsset(ctx context.Context, asset domain.ComponentAsset) (domain.ComponentAsset, error) {
	if err := r.store.db.QueryRowxContext(ctx, `
		update component_assets
		set source = $1, status = $2, label = $3, url_or_path = $4,
		    preview_url = $5, metadata_json = $6, updated_at = now()
		where id = $7
		returning created_at, updated_at, component_id, asset_type
	`, asset.Source, asset.Status, asset.Label, asset.URLOrPath,
		asset.PreviewURL, asset.MetadataJSON, asset.ID,
	).Scan(&asset.CreatedAt, &asset.UpdatedAt, &asset.ComponentID, &asset.AssetType); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ComponentAsset{}, domain.ErrNotFound{ID: asset.ID}
		}
		return domain.ComponentAsset{}, err
	}
	return asset, nil
}

func (r *ComponentAssetRepository) DeleteComponentAsset(ctx context.Context, id string) error {
	res, err := r.store.db.ExecContext(ctx, `delete from component_assets where id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: id}
	}
	return nil
}

func (r *ComponentAssetRepository) SetSelectedComponentAsset(ctx context.Context, componentID string, assetType domain.AssetType, assetID string) error {
	col, err := domain.SelectedColumnForType(assetType)
	if err != nil {
		return err
	}

	// Use a single statement that validates the asset belongs to the component
	// and matches the expected type. If no rows are updated, either the component
	// or the asset does not exist, or the asset does not match.
	query := fmt.Sprintf(`
		update components
		set %s = $1, updated_at = now()
		where id = $2
		  and exists (
		    select 1 from component_assets
		    where id = $1 and component_id = $2 and asset_type = $3
		  )
	`, col)

	res, err := r.store.db.ExecContext(ctx, query, assetID, componentID, assetType)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		// Determine why: component missing, asset missing, or ownership/type mismatch.
		var exists bool
		_ = r.store.db.QueryRowContext(ctx, `select exists(select 1 from components where id = $1)`, componentID).Scan(&exists)
		if !exists {
			return domain.ErrNotFound{ID: componentID}
		}
		var asset domain.ComponentAsset
		if err := r.store.db.GetContext(ctx, &asset, `
			select id, component_id, asset_type from component_assets where id = $1
		`, assetID); err != nil {
			return domain.ErrNotFound{ID: assetID}
		}
		if asset.ComponentID != componentID {
			return domain.ErrAssetNotOwned{AssetID: assetID, ComponentID: componentID}
		}
		return domain.ErrAssetTypeMismatch{AssetType: asset.AssetType, SlotType: assetType}
	}
	return nil
}

func (r *ComponentAssetRepository) ClearSelectedComponentAsset(ctx context.Context, componentID string, assetType domain.AssetType) error {
	col, err := domain.SelectedColumnForType(assetType)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		update components
		set %s = null, updated_at = now()
		where id = $1
	`, col)

	res, err := r.store.db.ExecContext(ctx, query, componentID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrNotFound{ID: componentID}
	}
	return nil
}

func (r *ComponentAssetRepository) GetComponentWithAssets(ctx context.Context, componentID string) (domain.ComponentWithAssets, error) {
	// Fetch the component (including selected asset IDs).
	var component domain.Component
	if err := r.store.db.GetContext(ctx, &component, `
		select id, category, mpn, manufacturer, package, description,
		       quantity, quantity_mode, location, created_at, updated_at,
		       selected_symbol_asset_id, selected_footprint_asset_id,
		       selected_3d_model_asset_id, selected_datasheet_asset_id
		from components
		where id = $1
	`, componentID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ComponentWithAssets{}, domain.ErrNotFound{ID: componentID}
		}
		return domain.ComponentWithAssets{}, err
	}

	// Fetch attributes.
	compRepo := &ComponentRepository{store: r.store}
	attrs, err := compRepo.getComponentAttributes(ctx, []string{componentID})
	if err != nil {
		return domain.ComponentWithAssets{}, err
	}
	component.Attributes = attrs[componentID]

	// Fetch all attached assets.
	assets, err := r.ListComponentAssets(ctx, componentID)
	if err != nil {
		return domain.ComponentWithAssets{}, err
	}

	result := domain.ComponentWithAssets{
		Component: component,
		Assets:    assets,
	}

	// Resolve selected assets by ID.
	assetByID := make(map[string]*domain.ComponentAsset, len(assets))
	for i := range assets {
		assetByID[assets[i].ID] = &assets[i]
	}
	if component.SelectedSymbolAssetID != nil {
		if a, ok := assetByID[*component.SelectedSymbolAssetID]; ok {
			result.SelectedSymbolAsset = a
		}
	}
	if component.SelectedFootprintAssetID != nil {
		if a, ok := assetByID[*component.SelectedFootprintAssetID]; ok {
			result.SelectedFootprintAsset = a
		}
	}
	if component.Selected3DModelAssetID != nil {
		if a, ok := assetByID[*component.Selected3DModelAssetID]; ok {
			result.Selected3DModelAsset = a
		}
	}
	if component.SelectedDatasheetAssetID != nil {
		if a, ok := assetByID[*component.SelectedDatasheetAssetID]; ok {
			result.SelectedDatasheetAsset = a
		}
	}

	return result, nil
}

var _ domain.ComponentAssetRepository = (*ComponentAssetRepository)(nil)
