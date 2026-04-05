package postgres

import (
	"context"
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func (s *Store) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrationFiles)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	return goose.UpContext(ctx, s.db.DB, "migrations")
}
