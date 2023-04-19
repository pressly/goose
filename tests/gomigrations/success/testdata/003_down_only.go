package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(nil, down003)
}

func down003(ctx context.Context, tx *sql.Tx) error {
	q := "TRUNCATE TABLE foo"
	_, err := tx.ExecContext(ctx, q)
	return err
}
