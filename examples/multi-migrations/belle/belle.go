package belle

import (
	"embed"

	"github.com/pressly/goose/v3"
)

//go:embed [0-9]*_*.*
var migrationsFS embed.FS

var Provider = goose.NewProvider(
	goose.ProviderPackage("belle", "Provider"),
	goose.Filesystem(migrationsFS),
	goose.Tablename("belle_db_version"),
	goose.Dialect(goose.DialectSQLite3),
	goose.BaseDir(""), // use the directory this package is in
)
