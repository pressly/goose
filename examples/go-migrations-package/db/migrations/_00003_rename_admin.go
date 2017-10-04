package migrations

import (
	"database/sql"

	"github.com/mc2soft/goose"
)

func init() {
	goose.AddMigration(Up00003, Down00003)
}

func Up00003(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='admin3' WHERE username='admin4';")
	if err != nil {
		return err
	}
	return nil
}

func Down00003(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='admin4' WHERE username='admin3';")
	if err != nil {
		return err
	}
	return nil
}
