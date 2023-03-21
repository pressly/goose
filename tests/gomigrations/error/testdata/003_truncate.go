package gomigrations

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(up003, nil)
}

func up003(tx *sql.Tx) error {
	q := "TRUNCATE TABLE foo"
	_, err := tx.Exec(q)
	return err
}
