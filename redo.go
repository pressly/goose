package goose

import (
	"context"
	"database/sql"
	"errors"

	"go.uber.org/multierr"
)

// Redo rolls back the most recently applied migration, then runs it again.
func Redo(db *sql.DB, dir string, opts ...OptionsFunc) (retErr error) {
	ctx := context.Background()

	option := &options{}
	for _, f := range opts {
		f(option)
	}
	migrations, err := CollectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
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
		}()
	case LockModeAdvisoryTransaction:
		return errors.New("advisory level transaction lock is not supported")
	}

	var (
		currentVersion int64
	)
	if option.noVersioning {
		if len(migrations) == 0 {
			return nil
		}
		currentVersion = migrations[len(migrations)-1].Version
	} else {
		if currentVersion, err = GetDBVersion(db); err != nil {
			return err
		}
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return err
	}
	current.noVersioning = option.noVersioning

	if err := current.Down(db); err != nil {
		return err
	}
	if err := current.Up(db); err != nil {
		return err
	}
	return nil
}
