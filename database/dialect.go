package database

import (
	"context"
	"github.com/pressly/goose/v3/internal/dialect"
	"github.com/pressly/goose/v3/migration"

	"github.com/pressly/goose/v3/internal/dialectstore"
)

// NewStore returns a new [Store] implementation for the given dialect.
func NewStore(d dialect.Dialect, tableName string) (Store, error) {
	dialectStore, err := dialectstore.NewStore(d, tableName)
	if err != nil {
		return nil, err
	}

	return &store{dialectStore: dialectStore}, nil
}

type store struct {
	dialectStore dialectstore.Store
}

var _ Store = (*store)(nil)

func (s *store) Tablename() string {
	return s.dialectStore.GetTableName()
}

func (s *store) CreateVersionTable(ctx context.Context, db DBTxConn) error {
	return s.dialectStore.CreateVersionTable(ctx, db)
}

func (s *store) Insert(ctx context.Context, db DBTxConn, req migration.Entity) error {
	return s.dialectStore.InsertVersion(ctx, db, req)
}

func (s *store) Delete(ctx context.Context, db DBTxConn, version int64) error {
	return s.dialectStore.DeleteVersion(ctx, db, migration.Entity{Version: version})
}

func (s *store) GetMigration(
	ctx context.Context,
	db DBTxConn,
	version int64,
) (*dialectstore.GetMigrationResult, error) {
	return s.dialectStore.GetMigration(ctx, db, version)
}

func (s *store) GetLatestVersion(ctx context.Context, db DBTxConn) (int64, error) {
	return s.dialectStore.GetLatestVersion(ctx, db)
}

func (s *store) ListMigrations(
	ctx context.Context,
	db DBTxConn,
) ([]*dialectstore.ListMigrationsResult, error) {
	return s.dialectStore.ListMigrations(ctx, db)
}

func (s *store) TableExists(ctx context.Context, db DBTxConn) (bool, error) {
	return s.dialectStore.TableVersionExists(ctx, db)
}
