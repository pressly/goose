package testdata

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// MustMigrationsFS returns the embedded migrations filesystem.
func MustMigrationsFS() fs.FS {
	fsys, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		// This should never happen, since the subdirectory is hardcoded. If the layout of the
		// embedded files changes, this will panic to alert the developer to update the code
		// accordingly.
		panic(err)
	}
	return fsys
}
