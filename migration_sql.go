package goose

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

const sqlCmdPrefix = "-- +goose "
const scanBufSize = 4 * 1024 * 1024

var bufferPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, scanBufSize)
	},
}

// Checks the line to see if the line has a statement-ending semicolon
// or if the line contains a double-dash comment.
func endsWithSemicolon(line string) bool {
	scanBuf := bufferPool.Get().([]byte)
	defer bufferPool.Put(scanBuf)

	prev := ""
	scanner := bufio.NewScanner(strings.NewReader(line))
	scanner.Buffer(scanBuf, scanBufSize)
	scanner.Split(bufio.ScanWords)

	for scanner.Scan() {
		word := scanner.Text()
		if strings.HasPrefix(word, "--") {
			break
		}
		prev = word
	}

	return strings.HasSuffix(prev, ";")
}

// Split the given sql script into individual statements.
//
// The base case is to simply split on semicolons, as these
// naturally terminate a statement.
//
// However, more complex cases like pl/pgsql can have semicolons
// within a statement. For these cases, we provide the explicit annotations
// 'StatementBegin' and 'StatementEnd' to allow the script to
// tell us to ignore semicolons.
func getSQLStatements(r io.Reader, direction bool) ([]string, bool, error) {
	var buf bytes.Buffer
	scanBuf := bufferPool.Get().([]byte)
	defer bufferPool.Put(scanBuf)

	scanner := bufio.NewScanner(r)
	scanner.Buffer(scanBuf, scanBufSize)

	// track the count of each section
	// so we can diagnose scripts with no annotations
	upSections := 0
	downSections := 0

	statementEnded := false
	ignoreSemicolons := false
	directionIsActive := false
	tx := true
	stmts := []string{}

	for scanner.Scan() {

		line := scanner.Text()

		// handle any goose-specific commands
		if strings.HasPrefix(line, sqlCmdPrefix) {
			cmd := strings.TrimSpace(line[len(sqlCmdPrefix):])
			switch cmd {
			case "Up":
				directionIsActive = (direction == true)
				upSections++
				break

			case "Down":
				directionIsActive = (direction == false)
				downSections++
				break

			case "StatementBegin":
				if directionIsActive {
					ignoreSemicolons = true
				}
				break

			case "StatementEnd":
				if directionIsActive {
					statementEnded = (ignoreSemicolons == true)
					ignoreSemicolons = false
				}
				break

			case "NO TRANSACTION":
				tx = false
				break
			}
		}

		if !directionIsActive {
			continue
		}

		if _, err := buf.WriteString(line + "\n"); err != nil {
			return nil, false, fmt.Errorf("io err: %v", err)
		}

		// Wrap up the two supported cases: 1) basic with semicolon; 2) psql statement
		// Lines that end with semicolon that are in a statement block
		// do not conclude statement.
		if (!ignoreSemicolons && endsWithSemicolon(line)) || statementEnded {
			statementEnded = false
			stmts = append(stmts, buf.String())
			buf.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, false, fmt.Errorf("scanning migration: %v", err)
	}

	// diagnose likely migration script errors
	if ignoreSemicolons {
		return nil, false, fmt.Errorf("parsing migration: saw '-- +goose StatementBegin' with no matching '-- +goose StatementEnd'")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		return nil, false, fmt.Errorf("parsing migration: unexpected unfinished SQL query: %s. potential missing semicolon", bufferRemaining)
	}

	if upSections == 0 && downSections == 0 {
		return nil, false, fmt.Errorf("parsing migration: no Up/Down annotations found, so no statements were executed. See https://bitbucket.org/liamstask/goose/overview for details")
	}

	return stmts, tx, nil
}

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

	statements, useTx, err := getSQLStatements(f, direction)
	if err != nil {
		return err
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
	matchEmptyLines  = regexp.MustCompile(`(?m)^$[\r\n]*`)
)

func clearStatement(s string) string {
	s = matchSQLComments.ReplaceAllString(s, ``)
	return matchEmptyLines.ReplaceAllString(s, ``)
}
