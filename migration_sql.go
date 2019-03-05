package goose

import (
	"database/sql"
	"os"
	"regexp"

	"github.com/pkg/errors"
)

// Run a migration specified in raw SQL.
//
// Sections of the script can be annotated with a special comment,
// starting with "-- +goose" to specify whether the section should
// be applied during an Up or Down migration
//
// All statements following an Up or Down directive are grouped together
// until another direction directive is found.
func runSQLMigration(db *sql.DB, sqlFile string, v int64, direction bool) error {
	f, err := os.Open(sqlFile)
	if err != nil {
		return errors.Wrap(err, "failed to open SQL migration file")
	}
	defer f.Close()

	statements, useTx, err := parseSQLMigrationFile(f, direction)
	if err != nil {
		return errors.Wrap(err, "failed to parse SQL migration file")
	}

	if useTx {
		// TRANSACTION.

		printInfo("Begin transaction\n")

		tx, err := db.Begin()
		if err != nil {
			errors.Wrap(err, "failed to begin transaction")
		}

		for _, query := range statements {
			printInfo("Executing statement: %s\n", clearStatement(query))
			if _, err = tx.Exec(query); err != nil {
				printInfo("Rollback transaction\n")
				tx.Rollback()
				return errors.Wrapf(err, "failed to execute SQL query %q", clearStatement(query))
			}
		}

		if direction {
			if _, err := tx.Exec(GetDialect().insertVersionSQL(), v, direction); err != nil {
				printInfo("Rollback transaction\n")
				tx.Rollback()
				return errors.Wrap(err, "failed to insert new goose version")
			}
		} else {
			if _, err := tx.Exec(GetDialect().deleteVersionSQL(), v); err != nil {
				printInfo("Rollback transaction\n")
				tx.Rollback()
				return errors.Wrap(err, "failed to delete goose version")
			}
		}

		printInfo("Commit transaction\n")
		if err := tx.Commit(); err != nil {
			return errors.Wrap(err, "failed to commit transaction")
		}

		return nil
	}

	// NO TRANSACTION.
	for _, query := range statements {
		printInfo("Executing statement: %s\n", clearStatement(query))
		if _, err := db.Exec(query); err != nil {
			return errors.Wrapf(err, "failed to execute SQL query %q", clearStatement(query))
		}
	}
	if _, err := db.Exec(GetDialect().insertVersionSQL(), v, direction); err != nil {
		return errors.Wrap(err, "failed to insert new goose version")
	}

	return nil
}

func printInfo(s string, args ...interface{}) {
	if verbose {
		log.Printf(s, args...)
	}
}

var (
	matchSQLComments = regexp.MustCompile(`(?m)^--.*$[\r\n]*`)
	matchEmptyLines  = regexp.MustCompile(`(?m)^$[\r\n]*`) // TODO: Duplicate
)

func clearStatement(s string) string {
	s = matchSQLComments.ReplaceAllString(s, ``)
	return matchEmptyLines.ReplaceAllString(s, ``)
}
