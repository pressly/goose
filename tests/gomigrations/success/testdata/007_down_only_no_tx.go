package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationNoTx(nil, down007)
}

func down007(ctx context.Context, db *sql.DB) error {
	q := "TRUNCATE TABLE users"
	_, err := db.ExecContext(ctx, q)
	return err
}
