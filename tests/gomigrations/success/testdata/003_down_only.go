package gomigrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(nil, down003)
}

func down003(tx *sql.Tx) error {
	q := "TRUNCATE TABLE foo"
	_, err := tx.Exec(q)
	return err
}
