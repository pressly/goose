package iofs

import (
	"database/sql"
	"io/fs"

	"github.com/pressly/goose"
)

// Down rolls back a single migration from the current version.
func Down(db *sql.DB, fsys fs.FS, dir string) error {
	migrations, err := CollectMigrations(fsys, dir, 0, goose.MaxVersion)
	if err != nil {
		return err
	}

	return migrations.Down(db)
}

// DownTo rolls back migrations to a specific version.
func DownTo(db *sql.DB, fsys fs.FS, dir string, version int64) error {
	migrations, err := CollectMigrations(fsys, dir, 0, goose.MaxVersion)
	if err != nil {
		return err
	}

	return migrations.DownTo(db, version)
}

