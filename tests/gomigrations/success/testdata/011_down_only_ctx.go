package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(nil, down011)
}

func down011(ctx context.Context, tx *sql.Tx) error {
	return dropTable(tx, "foxtrot")
}
