package gomigrations

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(nil, down007)
}

func down007(db *sql.DB) error {
	return dropTable(db, "delta")
}
