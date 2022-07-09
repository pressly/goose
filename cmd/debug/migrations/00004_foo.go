package migrations

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.Register(Up00002, Down00002)
}

func Up00002(tx *sql.Tx) error {
	return nil
}

func Down00002(tx *sql.Tx) error {
	return nil
}
