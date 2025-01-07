package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationNoTx(up001, nil)
}

func up001(ctx context.Context, db *sql.DB) error {
	q := "CREATE TABLE foo (id INT)"
	_, err := db.ExecContext(ctx, q)
	return err
}
