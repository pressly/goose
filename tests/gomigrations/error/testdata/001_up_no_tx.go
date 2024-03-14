package gomigrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(up001, nil)
}

func up001(db *sql.DB) error {
	q := "CREATE TABLE foo (id INTEGER)"
	_, err := db.Exec(q)
	return err
}
