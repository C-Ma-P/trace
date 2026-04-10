package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"trace/internal/domain"
)

type ComponentRepository struct {
	store *Store
}

func NewComponentRepository(store *Store) *ComponentRepository {
	return &ComponentRepository{store: store}
}

func (r *ComponentRepository) UpsertAttributeDefinition(ctx context.Context, def domain.AttributeDefinition) error {
	_, err := r.store.db.ExecContext(ctx, `
		insert into attribute_definitions(key, category, value_type, display_name, unit)
		values ($1, $2, $3, $4, $5)
		on conflict (key, category) do update
			set value_type   = excluded.value_type,
			    display_name = excluded.display_name,
			    unit         = excluded.unit
	`, def.Key, def.Category, def.ValueType, def.DisplayName, def.Unit)
	return err
}

func (r *ComponentRepository) CreateComponent(ctx context.Context, component domain.Component) (domain.Component, error) {
	tx, err := r.store.db.BeginTxx(ctx, nil)
	if err != nil {
		return domain.Component{}, err
	}
	defer tx.Rollback()

	if component.QuantityMode == "" {
		component.QuantityMode = domain.QuantityModeUnknown
	}

	if err := tx.QueryRowxContext(ctx, `
		insert into components(id, category, mpn, manufacturer, package, description, quantity, quantity_mode, location)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		returning created_at, updated_at
	`, component.ID, component.Category, component.MPN, component.Manufacturer, component.Package, component.Description,
		component.Quantity, component.QuantityMode, component.Location,
	).Scan(&component.CreatedAt, &component.UpdatedAt); err != nil {
		return domain.Component{}, err
	}

	for _, attribute := range component.Attributes {
		if err := validateAttributeValue(attribute); err != nil {
			return domain.Component{}, err
		}

		if _, err := tx.ExecContext(ctx, `
			insert into component_attributes(component_id, key, value_type, text_value, number_value, bool_value, unit)
			values ($1, $2, $3, $4, $5, $6, $7)
		`, component.ID, attribute.Key, attribute.ValueType, attribute.Text, attribute.Number, attribute.Bool, attribute.Unit); err != nil {
			return domain.Component{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return domain.Component{}, err
	}

	return component, nil
}

func (r *ComponentRepository) GetComponent(ctx context.Context, id string) (domain.Component, error) {
	var component domain.Component
	if err := r.store.db.GetContext(ctx, &component, `
		select id, category, mpn, manufacturer, package, description,
		       quantity, quantity_mode, location, created_at, updated_at,
		       selected_symbol_asset_id, selected_footprint_asset_id,
		       selected_3d_model_asset_id, selected_datasheet_asset_id
		from components
		where id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Component{}, domain.ErrNotFound{ID: id}
		}
		return domain.Component{}, err
	}

	attributes, err := r.getComponentAttributes(ctx, []string{id})
	if err != nil {
		return domain.Component{}, err
	}

	component.Attributes = attributes[id]
	return component, nil
}

func (r *ComponentRepository) ListComponentsByCategory(ctx context.Context, category domain.Category) ([]domain.Component, error) {
	var components []domain.Component
	if err := r.store.db.SelectContext(ctx, &components, `
		select id, category, mpn, manufacturer, package, description,
		       quantity, quantity_mode, location, created_at, updated_at,
		       selected_symbol_asset_id, selected_footprint_asset_id,
		       selected_3d_model_asset_id, selected_datasheet_asset_id
		from components
		where category = $1
		order by manufacturer, mpn
	`, category); err != nil {
		return nil, err
	}

	if len(components) == 0 {
		return components, nil
	}

	ids := make([]string, len(components))
	for i, c := range components {
		ids[i] = c.ID
	}

	attributesByID, err := r.getComponentAttributes(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range components {
		components[i].Attributes = attributesByID[components[i].ID]
	}

	return components, nil
}

func (r *ComponentRepository) getComponentAttributes(ctx context.Context, componentIDs []string) (map[string][]domain.AttributeValue, error) {
	query, args, err := sqlx.In(`
		select component_id, key, value_type, text_value, number_value, bool_value, unit
		from component_attributes
		where component_id in (?)
		order by key
	`, componentIDs)
	if err != nil {
		return nil, err
	}

	var rows []attributeRow
	if err := r.store.db.SelectContext(ctx, &rows, r.store.db.Rebind(query), args...); err != nil {
		return nil, err
	}

	result := make(map[string][]domain.AttributeValue, len(componentIDs))
	for _, row := range rows {
		result[row.ComponentID] = append(result[row.ComponentID], row.toAttributeValue())
	}
	return result, nil
}

func validateAttributeValue(attribute domain.AttributeValue) error {
	switch attribute.ValueType {
	case domain.ValueTypeText:
		if attribute.Text == nil {
			return fmt.Errorf("attribute %q requires text value", attribute.Key)
		}
	case domain.ValueTypeNumber:
		if attribute.Number == nil {
			return fmt.Errorf("attribute %q requires number value", attribute.Key)
		}
	case domain.ValueTypeBool:
		if attribute.Bool == nil {
			return fmt.Errorf("attribute %q requires bool value", attribute.Key)
		}
	default:
		return fmt.Errorf("attribute %q has unsupported value type %q", attribute.Key, attribute.ValueType)
	}

	return nil
}

func (r *ComponentRepository) UpdateComponentMetadata(ctx context.Context, component domain.Component) (domain.Component, error) {
	if err := r.store.db.QueryRowxContext(ctx, `
		update components
		set mpn = $1, manufacturer = $2, package = $3, description = $4, updated_at = now()
		where id = $5
		returning quantity, quantity_mode, location, created_at, updated_at,
		          selected_symbol_asset_id, selected_footprint_asset_id,
		          selected_3d_model_asset_id, selected_datasheet_asset_id
	`, component.MPN, component.Manufacturer, component.Package, component.Description, component.ID,
	).Scan(&component.Quantity, &component.QuantityMode, &component.Location, &component.CreatedAt, &component.UpdatedAt,
		&component.SelectedSymbolAssetID, &component.SelectedFootprintAssetID,
		&component.Selected3DModelAssetID, &component.SelectedDatasheetAssetID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Component{}, domain.ErrNotFound{ID: component.ID}
		}
		return domain.Component{}, err
	}
	return component, nil
}

func (r *ComponentRepository) UpdateComponentInventory(ctx context.Context, component domain.Component) (domain.Component, error) {
	if err := r.store.db.QueryRowxContext(ctx, `
		update components
		set quantity = $1, quantity_mode = $2, location = $3, updated_at = now()
		where id = $4
		returning mpn, manufacturer, package, description, created_at, updated_at,
		          selected_symbol_asset_id, selected_footprint_asset_id,
		          selected_3d_model_asset_id, selected_datasheet_asset_id
	`, component.Quantity, component.QuantityMode, component.Location, component.ID,
	).Scan(&component.MPN, &component.Manufacturer, &component.Package, &component.Description,
		&component.CreatedAt, &component.UpdatedAt,
		&component.SelectedSymbolAssetID, &component.SelectedFootprintAssetID,
		&component.Selected3DModelAssetID, &component.SelectedDatasheetAssetID,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Component{}, domain.ErrNotFound{ID: component.ID}
		}
		return domain.Component{}, err
	}
	return component, nil
}

func (r *ComponentRepository) ReplaceComponentAttributes(ctx context.Context, componentID string, attrs []domain.AttributeValue) error {
	tx, err := r.store.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		delete from component_attributes where component_id = $1
	`, componentID); err != nil {
		return err
	}

	for _, attr := range attrs {
		if err := validateAttributeValue(attr); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			insert into component_attributes(component_id, key, value_type, text_value, number_value, bool_value, unit)
			values ($1, $2, $3, $4, $5, $6, $7)
		`, componentID, attr.Key, attr.ValueType, attr.Text, attr.Number, attr.Bool, attr.Unit); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *ComponentRepository) FindComponents(ctx context.Context, filter domain.ComponentFilter) ([]domain.Component, error) {
	conditions := make([]string, 0, 5)
	args := make([]any, 0, 5)
	n := 1

	if filter.Category != nil {
		conditions = append(conditions, fmt.Sprintf("category = $%d", n))
		args = append(args, *filter.Category)
		n++
	}
	if filter.Manufacturer != "" {
		conditions = append(conditions, fmt.Sprintf("manufacturer ilike $%d", n))
		args = append(args, "%"+filter.Manufacturer+"%")
		n++
	}
	if filter.MPN != "" {
		conditions = append(conditions, fmt.Sprintf("mpn ilike $%d", n))
		args = append(args, "%"+filter.MPN+"%")
		n++
	}
	if filter.Package != "" {
		conditions = append(conditions, fmt.Sprintf("package = $%d", n))
		args = append(args, filter.Package)
		n++
	}
	if filter.Text != "" {
		conditions = append(conditions, fmt.Sprintf("(mpn ilike $%d or manufacturer ilike $%d or description ilike $%d)", n, n, n))
		args = append(args, "%"+filter.Text+"%")
		n++
	}

	query := `select id, category, mpn, manufacturer, package, description,
	       quantity, quantity_mode, location, created_at, updated_at,
	       selected_symbol_asset_id, selected_footprint_asset_id,
	       selected_3d_model_asset_id, selected_datasheet_asset_id from components`
	if len(conditions) > 0 {
		query += " where " + strings.Join(conditions, " and ")
	}
	query += " order by manufacturer, mpn"

	var components []domain.Component
	if err := r.store.db.SelectContext(ctx, &components, query, args...); err != nil {
		return nil, err
	}

	if len(components) == 0 {
		return components, nil
	}

	ids := make([]string, len(components))
	for i, c := range components {
		ids[i] = c.ID
	}

	attributesByID, err := r.getComponentAttributes(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i := range components {
		components[i].Attributes = attributesByID[components[i].ID]
	}

	return components, nil
}

func (r *ComponentRepository) DeleteComponent(ctx context.Context, id string) error {
	res, err := r.store.db.ExecContext(ctx, `delete from components where id = $1`, id)
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

var _ domain.ComponentRepository = (*ComponentRepository)(nil)
