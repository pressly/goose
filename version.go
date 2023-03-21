package goose

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/multierr"
)

// Version prints the current version of the database.
func Version(db *sql.DB, dir string, opts ...OptionsFunc) (retErr error) {
	ctx := context.Background()

	option := &options{}
	for _, f := range opts {
		f(option)
	}
	if option.noVersioning {
		var current int64
		migrations, err := CollectMigrations(dir, minVersion, maxVersion)
		if err != nil {
			return fmt.Errorf("failed to collect migrations: %w", err)
		}
		if len(migrations) > 0 {
			current = migrations[len(migrations)-1].Version
		}
		log.Printf("goose: file version %v\n", current)
		return nil
	}

	if option.lock {
		conn, err := db.Conn(ctx)
		if err != nil {
			return err
		}
		if err := store.LockSession(ctx, conn); err != nil {
			return err
		}
		defer func() {
			if err := store.UnlockSession(ctx, conn); err != nil {
				retErr = multierr.Append(retErr, err)
			}
		}()
	}

	current, err := GetDBVersion(db)
	if err != nil {
		return err
	}
	log.Printf("goose: version %v\n", current)
	return nil
}

var tableName = "goose_db_version"

// TableName returns goose db version table name
func TableName() string {
	return tableName
}

// SetTableName set goose db version table name
func SetTableName(n string) {
	tableName = n
}
