package register

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(
		func(_ context.Context, _ *sql.DB) error { return nil },
		func(_ context.Context, _ *sql.DB) error { return nil },
	)
}
