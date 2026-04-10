package postgres

import (
	"context"

	"trace/internal/domain"
)

type PreferenceRepository struct {
	store *Store
}

func NewPreferenceRepository(store *Store) *PreferenceRepository {
	return &PreferenceRepository{store: store}
}

func (r *PreferenceRepository) List(ctx context.Context, prefix string) (map[string]string, error) {
	rows := []struct {
		Name      string `db:"name"`
		ValueText string `db:"value_text"`
	}{}

	if err := r.store.db.SelectContext(ctx, &rows, `
		select name, value_text
		from app_preferences
		where name like $1
		order by name
	`, prefix+"%"); err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	for _, row := range rows {
		result[row.Name] = row.ValueText
	}
	return result, nil
}

func (r *PreferenceRepository) SetMany(ctx context.Context, values map[string]string) error {
	if len(values) == 0 {
		return nil
	}

	tx, err := r.store.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for key, value := range values {
		if _, err := tx.ExecContext(ctx, `
			insert into app_preferences(name, value_text)
			values ($1, $2)
			on conflict (name) do update
				set value_text = excluded.value_text,
				    updated_at = now()
		`, key, value); err != nil {
			return err
		}
	}

	return tx.Commit()
}

var _ domain.PreferenceRepository = (*PreferenceRepository)(nil)
