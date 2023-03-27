package gomigrations

import (
	"github.com/pressly/goose/v3/internal"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(nil, down007)
}

func down007(db internal.GooseDB) error {
	q := "TRUNCATE TABLE users"
	_, err := db.Exec(q)
	return err
}
