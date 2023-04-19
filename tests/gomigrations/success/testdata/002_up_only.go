package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(up002, nil)
}

func up002(ctx context.Context, tx *sql.Tx) error {
	q := "INSERT INTO foo VALUES (1, 1, 'Alice')"
	_, err := tx.ExecContext(ctx, q)
	return err
}
