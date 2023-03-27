package gomigrations

import (
	"github.com/pressly/goose/v3/internal"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(nil, down003)
}

func down003(tx internal.GooseTx) error {
	q := "TRUNCATE TABLE foo"
	_, err := tx.Exec(q)
	return err
}
