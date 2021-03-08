package goose

import (
	"context"
	"database/sql"
)

// Redo rolls back the most recently applied migration, then runs it again.
func RedoCtx(ctx context.Context, db *sql.DB, dir string) error {
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return err
	}

	if err := current.DownCtx(ctx, db); err != nil {
		return err
	}

	if err := current.UpCtx(ctx, db); err != nil {
		return err
	}

	return nil
}

// Redo rolls back the most recently applied migration, then runs it again.
func Redo(db *sql.DB, dir string) error {
	return RedoCtx(context.Background(), db, dir)
}
