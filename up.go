package goose

import (
	"context"
	"database/sql"
)

// UpToCtx migrates up to a specific version.
func UpToCtx(ctx context.Context, db *sql.DB, dir string, version int64) error {
	migrations, err := CollectMigrations(dir, minVersion, version)
	if err != nil {
		return err
	}

	for {
		current, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == ErrNoNextVersion {
				log.Printf("goose: no migrations to run. current version: %d\n", current)
				return nil
			}
			return err
		}

		if err = next.UpCtx(ctx, db); err != nil {
			return err
		}
	}
}

// UpTo migrates up to a specific version.
func UpTo(ctx context.Context, db *sql.DB, dir string, version int64) error {
	return UpToCtx(context.Background(), db, dir, version)
}

// UpCtx applies all available migrations.
func UpCtx(ctx context.Context, db *sql.DB, dir string) error {
	return UpTo(ctx, db, dir, maxVersion)
}

// Up applies all available migrations.
func Up(db *sql.DB, dir string) error {
	return UpCtx(context.Background(), db, dir)
}

// UpByOneCtx migrates up by a single version.
func UpByOneCtx(ctx context.Context, db *sql.DB, dir string) error {
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	next, err := migrations.Next(currentVersion)
	if err != nil {
		if err == ErrNoNextVersion {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
		}
		return err
	}

	if err = next.UpCtx(ctx, db); err != nil {
		return err
	}

	return nil
}

// UpByOne migrates up by a single version.
func UpByOne(ctx context.Context, db *sql.DB, dir string) error {
	return UpByOneCtx(context.Background(), db, dir)
}
