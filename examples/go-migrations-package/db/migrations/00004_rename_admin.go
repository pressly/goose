package migrations

import (
	"database/sql"

	"github.com/mc2soft/goose"
)

func init() {
	goose.AddMigration(Up00004, Down00004)
}

func Up00004(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='admin4' WHERE username='admin';")
	if err != nil {
		return err
	}
	return nil
}

func Down00004(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='admin' WHERE username='admin4';")
	if err != nil {
		return err
	}
	return nil
}
