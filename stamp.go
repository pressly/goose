package goose

import (
	"context"
	"database/sql"
	"fmt"
)

// StampTo stamps the database to a specific version.
func StampTo(db *sql.DB, dir string, version int64, opts ...OptionsFunc) error {
	ctx := context.Background()
	return StampToContext(ctx, db, dir, version, opts...)
}

func StampToContext(ctx context.Context, db *sql.DB, dir string, version int64, opts ...OptionsFunc) error {
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	foundMigrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		log.Printf("meh")
		return err
	}
	// Ensure that the target migration version actually exists.
	if _, err := foundMigrations.Current(version); err != nil && version != 0 {
		return fmt.Errorf("migration %d not found", version)
	}

	if _, err := EnsureDBVersionContext(ctx, db); err != nil {
		return err
	}
	dbMigrations, err := listAllDBVersions(ctx, db)
	if err != nil {
		return err
	}
	dbMaxVersion := dbMigrations[len(dbMigrations)-1].Version
	lookupAppliedInDB := make(map[int64]bool)
	for _, m := range dbMigrations {
		lookupAppliedInDB[m.Version] = true
	}

	if version < dbMaxVersion {
		// stamp "down"
		for {
			currentVersion, err := GetDBVersionContext(ctx, db)
			if err != nil {
				return err
			}

			if currentVersion == version {
				log.Printf("goose: database stamped to version: %d", currentVersion)
				return nil
			}
			_, err = foundMigrations.Current(currentVersion)
			if err != nil && !option.allowMissing {
				log.Printf("goose: migration file not found for current version (%d), error: %s", currentVersion, err)
				return err
			}

			if err := store.DeleteVersionNoTx(ctx, db, TableName(), currentVersion); err != nil {
				return fmt.Errorf("failed to delete goose version: %w", err)
			}
		}
	} else if version > dbMaxVersion {
		// stamp "up"
		for _, m := range foundMigrations {
			if lookupAppliedInDB[m.Version] {
				continue
			}
			if m.Version > dbMaxVersion && m.Version <= version {
				if err := store.InsertVersionNoTx(ctx, db, TableName(), m.Version); err != nil {
					return fmt.Errorf("failed to insert goose version: %w", err)
				}
			}
		}
		log.Printf("goose: database stamped to version: %d", version)
	} else {
		log.Printf("goose: database already at version: %d", version)
	}
	return nil
}

// Stamp stamps the database to the latest available migration.
func Stamp(db *sql.DB, dir string, opts ...OptionsFunc) error {
	ctx := context.Background()
	return StampContext(ctx, db, dir, opts...)
}

// StampContext stamps the database to the latest available migration.
func StampContext(ctx context.Context, db *sql.DB, dir string, opts ...OptionsFunc) error {
	return StampToContext(ctx, db, dir, maxVersion, opts...)
}
