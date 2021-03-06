package iofs

import (
	"database/sql"
	"io/fs"

	"github.com/pkg/errors"
	"github.com/pressly/goose"
)

// Reset rolls back all migrations
func Reset(db *sql.DB, fsys fs.FS, dir string) error {
	migrations, err := CollectMigrations(fsys, dir, 0, goose.MaxVersion)
	if err != nil {
		return errors.Wrap(err, "failed to collect migrations")
	}

	return migrations.Reset(db)
}
