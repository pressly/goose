package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(up001, down001)
}

func up001(ctx context.Context, tx *sql.Tx) error {
	q := "CREATE TABLE foo (id INT, subid INT, name TEXT)"
	_, err := tx.ExecContext(ctx, q)
	return err
}

func down001(ctx context.Context, tx *sql.Tx) error {
	q := "DROP TABLE IF EXISTS foo"
	_, err := tx.ExecContext(ctx, q)
	return err
}
