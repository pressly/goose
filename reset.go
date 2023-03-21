package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"go.uber.org/multierr"
)

// Reset rolls back all migrations
func Reset(db *sql.DB, dir string, opts ...OptionsFunc) (retErr error) {
	ctx := context.Background()
	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return fmt.Errorf("failed to collect migrations: %w", err)
	}

	switch option.lockMode {
	case LockModeAdvisorySession:
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
			if err := conn.Close(); err != nil {
				retErr = multierr.Append(retErr, err)
			}
		}()
	case LockModeAdvisoryTransaction:
		return errors.New("advisory level transaction lock is not supported")
	}

	if option.noVersioning {
		return DownTo(db, dir, minVersion, opts...)
	}

	statuses, err := dbMigrationsStatus(ctx, db)
	if err != nil {
		return fmt.Errorf("failed to get status of migrations: %w", err)
	}
	sort.Sort(sort.Reverse(migrations))

	for _, migration := range migrations {
		if !statuses[migration.Version] {
			continue
		}
		if err = migration.Down(db); err != nil {
			return fmt.Errorf("failed to db-down: %w", err)
		}
	}

	return nil
}

func dbMigrationsStatus(ctx context.Context, db *sql.DB) (map[int64]bool, error) {
	dbMigrations, err := store.ListMigrations(ctx, db)
	if err != nil {
		return nil, err
	}
	// The most recent record for each migration specifies
	// whether it has been applied or rolled back.
	results := make(map[int64]bool)

	for _, m := range dbMigrations {
		if _, ok := results[m.VersionID]; ok {
			continue
		}
		results[m.VersionID] = m.IsApplied
	}
	return results, nil
}
