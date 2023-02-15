package goose

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"
)

type migrateStatementSupported interface {
	migrateStatement(db *sql.DB, sql string) error
}

type insertVersionSupported interface {
	insertVersion(execer execer, versionID int64, isApplied bool) error
}

type deleteVersionSupported interface {
	deleteVersion(execer execer, versionID int64) error
}

// Run a migration specified in raw SQL.
//
// Sections of the script can be annotated with a special comment,
// starting with "-- +goose" to specify whether the section should
// be applied during an Up or Down migration
//
// All statements following an Up or Down directive are grouped together
// until another direction directive is found.
func runSQLMigration(db *sql.DB, statements []string, useTx bool, v int64, direction bool, noVersioning bool) error {
	if useTx {
		// TRANSACTION.

		verboseInfo("Begin transaction")

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}

		for _, query := range statements {
			verboseInfo("Executing statement: %s\n", clearStatement(query))
			if err = execQuery(tx.Exec, query); err != nil {
				verboseInfo("Rollback transaction")
				_ = tx.Rollback()
				return fmt.Errorf("failed to execute SQL query %q: %w", clearStatement(query), err)
			}
		}

		if !noVersioning {
			if direction {
				if d, ok := GetDialect().(insertVersionSupported); ok {
					err = d.insertVersion(tx, v, direction)
				} else {
					err = execQuery(tx.Exec, GetDialect().insertVersionSQL(), v, direction)
				}

				if err != nil {
					verboseInfo("Rollback transaction")
					_ = tx.Rollback()
					return fmt.Errorf("failed to insert new goose version: %w", err)
				}
			} else {
				if d, ok := GetDialect().(deleteVersionSupported); ok {
					err = d.deleteVersion(tx, v)
				} else {
					err = execQuery(tx.Exec, GetDialect().deleteVersionSQL(), v)
				}

				if err != nil {
					verboseInfo("Rollback transaction")
					_ = tx.Rollback()
					return fmt.Errorf("failed to delete goose version: %w", err)
				}
			}
		}

		verboseInfo("Commit transaction")
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		return nil
	}

	var err error

	// NO TRANSACTION.
	for _, query := range statements {
		verboseInfo("Executing statement: %s", clearStatement(query))

		if d, ok := GetDialect().(migrateStatementSupported); ok {
			err = d.migrateStatement(db, query)
		} else {
			err = execQuery(db.Exec, query)
		}

		if err != nil {
			return fmt.Errorf("failed to execute SQL query %q: %w", clearStatement(query), err)
		}
	}
	if !noVersioning {
		if direction {
			if d, ok := GetDialect().(insertVersionSupported); ok {
				err = d.insertVersion(db, v, direction)
			} else {
				err = execQuery(db.Exec, GetDialect().insertVersionSQL(), v, direction)
			}

			if err != nil {
				return fmt.Errorf("failed to insert new goose version: %w", err)
			}
		} else {
			if d, ok := GetDialect().(deleteVersionSupported); ok {
				err = d.deleteVersion(db, v)
			} else {
				err = execQuery(db.Exec, GetDialect().deleteVersionSQL(), v)
			}

			if err != nil {
				return fmt.Errorf("failed to delete goose version: %w", err)
			}
		}
	}

	return nil
}

func execQuery(fn func(string, ...interface{}) (sql.Result, error), query string, args ...interface{}) error {
	if !verbose {
		_, err := fn(query, args...)
		return err
	}

	ch := make(chan error)

	go func() {
		_, err := fn(query, args...)
		ch <- err
	}()

	t := time.Now()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case err := <-ch:
			return err
		case <-ticker.C:
			verboseInfo("Executing statement still in progress for %v", time.Since(t).Round(time.Second))
		}
	}
}

const (
	grayColor  = "\033[90m"
	resetColor = "\033[00m"
)

func verboseInfo(s string, args ...interface{}) {
	if verbose {
		if noColor {
			log.Printf(s, args...)
		} else {
			log.Printf(grayColor+s+resetColor, args...)
		}
	}
}

var (
	matchSQLComments = regexp.MustCompile(`(?m)^--.*$[\r\n]*`)
	matchEmptyEOL    = regexp.MustCompile(`(?m)^$[\r\n]*`) // TODO: Duplicate
)

func clearStatement(s string) string {
	s = matchSQLComments.ReplaceAllString(s, ``)
	return matchEmptyEOL.ReplaceAllString(s, ``)
}
