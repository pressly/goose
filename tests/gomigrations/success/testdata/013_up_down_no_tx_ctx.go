package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(up013, down013)
}

func up013(ctx context.Context, db *sql.DB) error {
	return createTable(db, "golf")
}

func down013(ctx context.Context, db *sql.DB) error {
	return dropTable(db, "golf")
}
