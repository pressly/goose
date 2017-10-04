package migrations

import (
	"database/sql"

	"github.com/mc2soft/goose"
)

func init() {
	goose.AddMigration(Up00005, Down00005)
}

func Up00005(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='admin5' WHERE username='admin3';")
	if err != nil {
		return err
	}
	return nil
}

func Down00005(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='admin3' WHERE username='admin5';")
	if err != nil {
		return err
	}
	return nil
}
