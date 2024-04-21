package database

import (
	"context"
	"database/sql"
	"errors"
)

// ErrNotSupported is returned when an optional method is not supported by the Store implementation.
var ErrNotSupported = errors.New("not supported")

// A StoreController is used by the goose package to interact with a database. This type is a
// wrapper around the Store interface, but can be extended to include additional (optional) methods
// that are not part of the core Store interface.
type StoreController struct {
	store Store
}

var _ Store = (*StoreController)(nil)

// NewStoreController returns a new StoreController that wraps the given Store.
//
// If the Store implements the following optional methods, the StoreController will call them as
// appropriate:
//
//   - TableExists(context.Context, DBTxConn) (bool, error)
//
// If the Store does not implement a method, it will either return a [ErrNotSupported] error or fall
// back to the default behavior.
func NewStoreController(store Store) *StoreController {
	return &StoreController{store: store}
}

// TableExists is an optional method that checks if the version table exists in the database. It is
// recommended to implement this method if the database supports it, as it can be used to optimize
// certain operations.
func (c *StoreController) TableExists(ctx context.Context, db *sql.Conn) (bool, error) {
	if t, ok := c.store.(interface {
		TableExists(ctx context.Context, db *sql.Conn) (bool, error)
	}); ok {
		return t.TableExists(ctx, db)
	}
	return false, ErrNotSupported
}

// Default methods

func (c *StoreController) Tablename() string {
	return c.store.Tablename()
}

func (c *StoreController) CreateVersionTable(ctx context.Context, db DBTxConn) error {
	return c.store.CreateVersionTable(ctx, db)
}

func (c *StoreController) Insert(ctx context.Context, db DBTxConn, req InsertRequest) error {
	return c.store.Insert(ctx, db, req)
}

func (c *StoreController) Delete(ctx context.Context, db DBTxConn, version int64) error {
	return c.store.Delete(ctx, db, version)
}

func (c *StoreController) GetMigration(ctx context.Context, db DBTxConn, version int64) (*GetMigrationResult, error) {
	return c.store.GetMigration(ctx, db, version)
}

func (c *StoreController) GetLatestVersion(ctx context.Context, db DBTxConn) (int64, error) {
	return c.store.GetLatestVersion(ctx, db)
}

func (c *StoreController) ListMigrations(ctx context.Context, db DBTxConn) ([]*ListMigrationsResult, error) {
	return c.store.ListMigrations(ctx, db)
}
