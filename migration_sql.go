package goose

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"
)

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
				tx.Rollback()
				return fmt.Errorf("failed to execute SQL query %q: %w", clearStatement(query), err)
			}
		}

		if !noVersioning {
			if direction {
				if err := execQuery(tx.Exec, GetDialect().insertVersionSQL(), v, direction); err != nil {
					verboseInfo("Rollback transaction")
					tx.Rollback()
					return fmt.Errorf("failed to insert new goose version: %w", err)
				}
			} else {
				if err := execQuery(tx.Exec, GetDialect().deleteVersionSQL(), v); err != nil {
					verboseInfo("Rollback transaction")
					tx.Rollback()
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

	// NO TRANSACTION.
	for _, query := range statements {
		verboseInfo("Executing statement: %s", clearStatement(query))
		if err := execQuery(db.Exec, query); err != nil {
			return fmt.Errorf("failed to execute SQL query %q: %w", clearStatement(query), err)
		}
	}
	if !noVersioning {
		if direction {
			if err := execQuery(db.Exec, GetDialect().insertVersionSQL(), v, direction); err != nil {
				return fmt.Errorf("failed to insert new goose version: %w", err)
			}
		} else {
			if err := execQuery(db.Exec, GetDialect().deleteVersionSQL(), v); err != nil {
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

	for {
		select {
		case err := <-ch:
			return err
		case <-time.Tick(time.Minute):
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
		log.Printf(grayColor+s+resetColor, args...)
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
