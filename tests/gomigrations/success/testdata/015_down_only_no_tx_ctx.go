package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(nil, down015)
}

func down015(ctx context.Context, db *sql.DB) error {
	return dropTable(db, "hotel")
}
