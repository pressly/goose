package goose

import (
	"bufio"
	"bytes"
	"database/sql"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const sqlCmdPrefix = "-- +goose "

// Checks the line to see if the line has a statement-ending semicolon
// or if the line contains a double-dash comment.
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
		log.Fatalf("scanning migration: %v", err)
	}

	// diagnose likely migration script errors
	if ignoreSemicolons {
		log.Println("WARNING: saw '-- +goose StatementBegin' with no matching '-- +goose StatementEnd'")
	}

	if bufferRemaining := strings.TrimSpace(buf.String()); len(bufferRemaining) > 0 {
		log.Printf("WARNING: Unexpected unfinished SQL query: %s. Missing a semicolon?\n", bufferRemaining)
	}

	if upSections == 0 && downSections == 0 {
		log.Fatalf(`ERROR: no Up/Down annotations found, so no statements were executed.
			See https://bitbucket.org/liamstask/goose/overview for details.`)
	}

	return
}

func useTransactions(scriptFile string) bool {
	f, err := os.Open(scriptFile)
	if err != nil {
		log.Fatal(err)
	}

	noTransactionsRegex, _ := regexp.Compile("--\\s\\+goose\\sNO\\sTRANSACTION")

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()

		if noTransactionsRegex.MatchString(line) {
			f.Close()
			return false
		}
	}

	f.Close()
	return true
}

// Run a migration specified in raw SQL.
//
// Sections of the script can be annotated with a special comment,
// starting with "-- +goose" to specify whether the section should
// be applied during an Up or Down migration
//
// All statements following an Up or Down directive are grouped together
// until another direction directive is found.
func runSQLMigration(db *sql.DB, scriptFile string, v int64, direction bool) error {
	filePath := filepath.Base(scriptFile)
	useTx := useTransactions(scriptFile)

	f, err := os.Open(scriptFile)
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	if useTx {
		err := runMigrationInTransaction(db, f, v, direction, filePath)
		if err != nil {
			log.Fatalf("FAIL (tx) %s (%v), quitting migration.", filePath, err)
		}
	} else {
		err = runMigrationWithoutTransaction(db, f, v, direction, filePath)
		if err != nil {
			log.Fatalf("FAIL (no tx) %s (%v), quitting migration.", filePath, err)
		}
	}

	f.Close()

	return nil
}

// Run the migration within a transaction (recommended)
func runMigrationInTransaction(db *sql.DB, r io.Reader, v int64, direction bool, filePath string) error {
	txn, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}

	// find each statement, checking annotations for up/down direction
	// Commits the transaction if successfully applied each statement and
	// records the version into the version table or returns an error and
	// rolls back the transaction.
	for _, query := range splitSQLStatements(r, direction) {
		if _, err = txn.Exec(query); err != nil {
			txn.Rollback()
			return err
		}
	}

	if err = FinalizeMigrationTx(txn, direction, v); err != nil {
		log.Fatalf("error finalizing migration %s, quitting. (%v)", filePath, err)
	}

	return nil
}

func runMigrationWithoutTransaction(db *sql.DB, r io.Reader, v int64, direction bool, filePath string) error {
	// find each statement, checking annotations for up/down direction
	// Tecords the version into the version table or returns an error
	for _, query := range splitSQLStatements(r, direction) {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	if err := FinalizeMigration(db, direction, v); err != nil {
		log.Fatalf("error finalizing migration %s, quitting. (%v)", filePath, err)
	}

	return nil
}
