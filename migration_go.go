package goose

import (
	"database/sql"
	"fmt"
)

// Run a migration specified in Go using a transaction.
func runGoMigration(db *sql.DB, fn func(*sql.Tx) error, v int64, direction bool, noVersioning bool) error {
	verboseInfo("Begin transaction")
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ERROR failed to begin transaction: %w", err)
	}

	if fn != nil {
		// Run Go migration function.
		if err := fn(tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("ERROR failed to run Go migration function %T: %w", fn, err)
		}
	}
	if !noVersioning {
		if direction {
			if _, err := tx.Exec(GetDialect().insertVersionSQL(), v, direction); err != nil {
				tx.Rollback()
				return fmt.Errorf("ERROR failed to execute transaction: %w", err)
			}
		} else {
			if _, err := tx.Exec(GetDialect().deleteVersionSQL(), v); err != nil {
				tx.Rollback()
				return fmt.Errorf("ERROR failed to execute transaction: %w", err)
			}
		}
	}

	verboseInfo("Commit transaction")
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Run a migration specified in Go without using a transaction.
func runGoMigrationNoTx(db *sql.DB, fn func(*sql.DB) error, v int64, direction bool, noVersioning bool) error {
	if fn != nil {
		// Run Go migration function.
		if err := fn(db); err != nil {
			return fmt.Errorf("ERROR failed to run Go migration function %T: %w", fn, err)
		}
	}
	if !noVersioning {
		if direction {
			if _, err := db.Exec(GetDialect().insertVersionSQL(), v, direction); err != nil {
				return fmt.Errorf("ERROR failed to insert new goose version: %w", err)
			}
		} else {
			if _, err := db.Exec(GetDialect().deleteVersionSQL(), v); err != nil {
				return fmt.Errorf("ERROR failed to delete goose version: %w", err)
			}
		}
	}

	return nil
}
