package gomigrations

import (
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(up001, nil)
}

func up001(db goose.Connection) error {
	q := "CREATE TABLE foo (id INT)"
	_, err := db.Exec(q)
	return err
}
