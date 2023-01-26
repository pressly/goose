package gomigrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(up005, down005)
}

func up005(db *sql.DB) error {
	q := "CREATE TABLE users (id INT, email TEXT)"
	_, err := db.Exec(q)
	return err
}

func down005(db *sql.DB) error {
	q := "DROP TABLE IF EXISTS users"
	_, err := db.Exec(q)
	return err
}
