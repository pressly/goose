package gomigrations

import (
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(up003, nil)
}

func up003(tx goose.Tx) error {
	q := "TRUNCATE TABLE foo"
	_, err := tx.Exec(q)
	return err
}
