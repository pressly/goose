package main

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(Up00002, Down00002)
}

func Up00002(ctx context.Context, db *sql.DB) error {
	// This migration adds a full_name column to the users table
	_, err := db.ExecContext(ctx, `
		ALTER TABLE users
		ADD COLUMN full_name VARCHAR(100) NULL AFTER email
	`)
	return err
}

func Down00002(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
		ALTER TABLE users
		DROP COLUMN full_name
	`)
	return err
}
