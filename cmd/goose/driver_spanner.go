//go:build !no_spanner
// +build !no_spanner

package main

import (
	"database/sql"
	"fmt"

	_ "github.com/googleapis/go-sql-spanner"
)

func connect(projectId, instanceId, databaseId string) error {
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectId, instanceId, databaseId)
	db, err := sql.Open("spanner", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %v", err)
	}
	defer func() { _ = db.Close() }()

	fmt.Printf("Connected to %s\n", dsn)

	return nil
}
