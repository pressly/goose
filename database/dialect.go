package database

import (
	"context"
	"errors"
	"github.com/pressly/goose/v3/internal/dialect"
	"github.com/pressly/goose/v3/migration"

	"github.com/pressly/goose/v3/internal/dialectstore"
)

// NewStore returns a new [Store] implementation for the given dialect.
func NewStore(d dialect.Dialect, tablename string) (Store, error) {
	if tablename == "" {
		return nil, errors.New("table name must not be empty")
	}

	var result = &store{
		tablename: tablename,
	}

	if dialectStore, err := dialectstore.NewStore(d); err != nil {
		return nil, err
	} else {
		result.dialectStore = dialectStore
	}

	return result, nil
}

type store struct {
	tablename    string
	dialectStore dialectstore.Store
}

var _ Store = (*store)(nil)

func (s *store) Tablename() string {
	return s.tablename
}

func (s *store) CreateVersionTable(ctx context.Context, db DBTxConn) error {
	return s.dialectStore.CreateVersionTable(ctx, db, s.tablename)
}

func (s *store) Insert(ctx context.Context, db DBTxConn, req migration.Entity) error {
	return s.dialectStore.InsertVersion(ctx, db, s.tablename, req)
}

func (s *store) Delete(ctx context.Context, db DBTxConn, version int64) error {
	return s.dialectStore.DeleteVersion(ctx, db, s.tablename, migration.Entity{Version: version})
}

func (s *store) GetMigration(
	ctx context.Context,
	db DBTxConn,
	version int64,
) (*dialectstore.GetMigrationResult, error) {
	return s.dialectStore.GetMigration(ctx, db, s.tablename, version)
}

func (s *store) GetLatestVersion(ctx context.Context, db DBTxConn) (int64, error) {
	return s.dialectStore.GetLatestVersion(ctx, db, s.tablename)
}

func (s *store) ListMigrations(
	ctx context.Context,
	db DBTxConn,
) ([]*dialectstore.ListMigrationsResult, error) {
	return s.dialectStore.ListMigrations(ctx, db, s.tablename)
}

func (s *store) TableExists(ctx context.Context, db DBTxConn) (bool, error) {
	return s.dialectStore.TableVersionExists(ctx, db, s.tablename)
}
