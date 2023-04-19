package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationNoTx(up006, nil)
}

func up006(ctx context.Context, db *sql.DB) error {
	q := "INSERT INTO users VALUES (1, 'admin@example.com')"
	_, err := db.ExecContext(ctx, q)
	return err
}
