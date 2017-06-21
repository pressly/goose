package goose

import (
	"bufio"
	"bytes"
	"database/sql"
	"io"
	"log"
	"os"
	"strings"
)

// Run SQL statements defined in a given file for a given direction (Up or Down).
func runSQLMigration(db *sql.DB, scriptFile string, v int64, direction bool) error {
	f, err := os.Open(scriptFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	query, useTx := getSQLQuery(f, direction)

	if useTx {
		// TRANSACTION.

		tx, err := db.Begin()
		if err != nil {
			log.Fatal(err)
		}

		if _, err = tx.Exec(query); err != nil {
			tx.Rollback()
			return err
		}

		if _, err := tx.Exec(GetDialect().insertVersionSQL(), v, direction); err != nil {
			tx.Rollback()
			return err
		}

		return tx.Commit()
	}

	// NO TRANSACTION.
	if _, err := db.Exec(query); err != nil {
		return err
	}
	if _, err := db.Exec(GetDialect().insertVersionSQL(), v, direction); err != nil {
		return err
	}

	return nil
}

// Get SQL statements defined in a given file for a given direction (Up or Down).
func getSQLQuery(r io.Reader, direction bool) (statements string, tx bool) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(r)

	upSections := 0
	downSections := 0
	directionIsActive := false
	tx = true

	for scanner.Scan() {

		line := scanner.Text()

		// handle any goose-specific commands
		switch strings.ToLower(line) {
		case "-- +goose up":
			directionIsActive = (direction == true)
			upSections++
			break

		case "-- +goose down":
			directionIsActive = (direction == false)
			downSections++
			break

		case "-- +goose no transaction":
			if directionIsActive {
				log.Fatal("-- +goose NO TRANSACTION should be defined on top of the file")
			}
			tx = false
			break
		}

		if !directionIsActive {
			continue
		}

		if _, err := buf.WriteString(line + "\n"); err != nil {
			log.Fatalf("io err: %v", err)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("scanning migration: %v", err)
	}

	if upSections == 0 && downSections == 0 {
		log.Fatalf(`ERROR: No Up/Down annotations found.`)
	}

	return
}
