package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"
)

// Run a migration specified in raw SQL.
//
// Sections of the script can be annotated with a special comment,
// starting with "-- +goose" to specify whether the section should
// be applied during an Up or Down migration
//
// All statements following an Up or Down directive are grouped together
// until another direction directive is found.
func runSQLMigration(db *sql.DB, script string, v int, direction bool) (count int, err error) {

	txn, err := db.Begin()
	if err != nil {
		log.Fatal("db.Begin:", err)
	}

	f, err := ioutil.ReadFile(script)
	if err != nil {
		log.Fatal(err)
	}

	// ensure we don't apply a query until we're sure it's going
	// in the direction we're interested in
	directionIsActive := false
	count = 0

	// find each statement, checking annotations for up/down direction
	// and execute each of them in the current transaction
	stmts := strings.Split(string(f), ";")

	for _, query := range stmts {

		query = strings.TrimSpace(query)

		if strings.HasPrefix(query, "-- +goose Up") {
			directionIsActive = direction == true
		} else if strings.HasPrefix(query, "-- +goose Down") {
			directionIsActive = direction == false
		}

		if !directionIsActive || query == "" {
			continue
		}

		if _, err = txn.Exec(query); err != nil {
			txn.Rollback()
			log.Fatalf("FAIL %s (%v), quitting migration.", path.Base(script), err)
			return count, err
		}

		count++
	}

	if err = finalizeMigration(txn, direction, v); err != nil {
		log.Fatalf("error finalizing migration %s, quitting. (%v)", path.Base(script), err)
	}

	return count, nil
}

// Update the version table for the given migration,
// and finalize the transaction.
func finalizeMigration(txn *sql.Tx, direction bool, v int) error {

	if direction == false {
		v--
	}

	// XXX: drop goose_db_version table on some minimum version number?
	versionStmt := fmt.Sprintf("INSERT INTO goose_db_version (version_id) VALUES (%d);", v)
	if _, err := txn.Exec(versionStmt); err != nil {
		txn.Rollback()
		return err
	}

	return txn.Commit()
}
