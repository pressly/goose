package iofs

import (
	"database/sql"
	"io/fs"

	"github.com/pkg/errors"
	"github.com/pressly/goose"
)

// Status prints the status of all migrations.
func Status(db *sql.DB, fsys fs.FS, dir string) error {
	// collect all migrations
	migrations, err := CollectMigrations(fsys, dir, 0, goose.MaxVersion)
	if err != nil {
		return errors.Wrap(err, "failed to collect migrations")
	}

	return migrations.Status(db)
}

