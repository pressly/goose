package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationNoTx(up005, down005)
}

func up005(ctx context.Context, db *sql.DB) error {
	q := "CREATE TABLE users (id INT, email TEXT)"
	_, err := db.ExecContext(ctx, q)
	return err
}

func down005(ctx context.Context, db *sql.DB) error {
	q := "DROP TABLE IF EXISTS users"
	_, err := db.ExecContext(ctx, q)
	return err
}
