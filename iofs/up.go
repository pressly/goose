package iofs

import (
	"database/sql"
	"io/fs"

	"github.com/pressly/goose"
)

// UpTo migrates up to a specific version.
func UpTo(db *sql.DB, fsys fs.FS, dir string, version int64) error {
	migrations, err := CollectMigrations(fsys, dir, 0, version)
	if err != nil {
		return err
	}

	return migrations.Up(db)
}

// Up applies all available migrations.
func Up(db *sql.DB, fsys fs.FS, dir string) error {
	return UpTo(db, fsys, dir, goose.MaxVersion)
}

// UpByOne migrates up by a single version.
func UpByOne(db *sql.DB, fsys fs.FS, dir string) error {
	migrations, err := CollectMigrations(fsys, dir, 0, goose.MaxVersion)
	if err != nil {
		return err
	}

	return migrations.UpByOne(db)
}

