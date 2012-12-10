package main

import (
	"database/sql"
	"io/ioutil"
	"log"
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
func runSQLMigration(txn *sql.Tx, path string, v int, direction bool) (count int, err error) {

	f, err := ioutil.ReadFile(path)
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
			log.Println("error executing query:\n", query)
			return count, err
		}

		count++
	}

	return count, nil
}
