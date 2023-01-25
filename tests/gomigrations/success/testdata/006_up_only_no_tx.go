package gomigrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(up006, nil)
}

func up006(db *sql.DB) error {
	q := "INSERT INTO users VALUES (1, 'admin@example.com')"
	_, err := db.Exec(q)
	return err
}
