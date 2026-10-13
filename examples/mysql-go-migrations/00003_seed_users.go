package main

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(Up00003, Down00003)
}

func Up00003(ctx context.Context, tx *sql.Tx) error {
	// Seed initial users
	_, err := tx.ExecContext(ctx, `
		INSERT INTO users (username, email, full_name) VALUES
			('admin', 'admin@example.com', 'Administrator'),
			('john_doe', 'john@example.com', 'John Doe'),
			('jane_smith', 'jane@example.com', 'Jane Smith')
		ON DUPLICATE KEY UPDATE
			email = VALUES(email),
			full_name = VALUES(full_name)
	`)
	return err
}

func Down00003(ctx context.Context, tx *sql.Tx) error {
	// Remove seeded users
	_, err := tx.ExecContext(ctx, `
		DELETE FROM users WHERE username IN ('admin', 'john_doe', 'jane_smith')
	`)
	return err
}
