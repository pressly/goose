package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(up014, nil)
}

func up014(ctx context.Context, db *sql.DB) error {
	return createTable(db, "hotel")
}
