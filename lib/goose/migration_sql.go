package goose

import (
	"bufio"
	"bytes"
	"database/sql"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const sqlCmdPrefix = "-- +goose "

func endsWithSemicolon(line string) bool {

	prev := ""
	scanner := bufio.NewScanner(strings.NewReader(line))
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
func splitSQLStatements(r io.Reader, direction bool) (stmts []string) {

	var buf bytes.Buffer
	scanner := bufio.NewScanner(r)

	// track the count of each section
	// so we can diagnose scripts with no annotations
	upSections := 0
	downSections := 0

	statementEnded := false
	ignoreSemicolons := false
	directionIsActive := false

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
			}
		}

		if !directionIsActive {
			continue
		}

		if _, err := buf.WriteString(line + "\n"); err != nil {
			log.Fatalf("io err: %v", err)
		}

		if !ignoreSemicolons && (statementEnded || endsWithSemicolon(line)) {
			statementEnded = false
			stmts = append(stmts, buf.String())
			buf.Reset()
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("scanning migration: %v", err)
	}

	// diagnose likely migration script errors
	if ignoreSemicolons {
		log.Println("WARNING: saw '-- +goose StatementBegin' with no matching '-- +goose StatementEnd'")
	}

	if upSections == 0 && downSections == 0 {
		log.Fatalf(`ERROR: no Up/Down annotations found, so no statements were executed.
			See https://bitbucket.org/liamstask/goose/overview for details.`)
	}

	return
}

// Run a migration specified in raw SQL.
//
// Sections of the script can be annotated with a special comment,
// starting with "-- +goose" to specify whether the section should
// be applied during an Up or Down migration
//
// All statements following an Up or Down directive are grouped together
// until another direction directive is found.
func runSQLMigration(conf *DBConf, db *sql.DB, script string, v int64, direction bool) error {

	txn, err := db.Begin()
	if err != nil {
		log.Fatal("db.Begin:", err)
	}

	f, err := os.Open(script)
	if err != nil {
		log.Fatal(err)
	}

	// find each statement, checking annotations for up/down direction
	// and execute each of them in the current transaction
	for _, query := range splitSQLStatements(f, direction) {
		if _, err = txn.Exec(query); err != nil {
			txn.Rollback()
			log.Fatalf("FAIL %s (%v), quitting migration.", filepath.Base(script), err)
			return err
		}
	}

	if err = finalizeMigration(conf, txn, direction, v); err != nil {
		log.Fatalf("error finalizing migration %s, quitting. (%v)", filepath.Base(script), err)
	}

	return nil
}

// Update the version table for the given migration,
// and finalize the transaction.
func finalizeMigration(conf *DBConf, txn *sql.Tx, direction bool, v int64) error {

	// XXX: drop goose_db_version table on some minimum version number?
	d := conf.Driver.Dialect
	if _, err := txn.Exec(d.insertVersionSql(), v, direction); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}
