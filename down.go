package goose

import (
	"context"
	"database/sql"
	"fmt"
)

// DownCtx rolls back a single migration from the current version.
func DownCtx(ctx context.Context, db *sql.DB, dir string) error {
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
		return fmt.Errorf("no migration %v", currentVersion)
	}

	return current.DownCtx(ctx, db)
}

// Down rolls back a single migration from the current version.
func Down(db *sql.DB, dir string) error {
	return DownCtx(context.Background(), db, dir)
}

// DownToCtx rolls back migrations to a specific version.
func DownToCtx(ctx context.Context, db *sql.DB, dir string, version int64) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	for {
		currentVersion, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		current, err := migrations.Current(currentVersion)
		if err != nil {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}

		if current.Version <= version {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
			return nil
		}

		if err = current.DownCtx(ctx, db); err != nil {
			return err
		}
	}
}

// DownTo rolls back migrations to a specific version.
func DownTo(db *sql.DB, dir string, version int64) error {
	return DownToCtx(context.Background(), db, dir, version)
}
