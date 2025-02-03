package gomigrations

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationContext(up010, nil)
}

func up010(ctx context.Context, tx *sql.Tx) error {
	return createTable(tx, "foxtrot")
}
