package testdata

//go:generate sha256sum $(find ./migrations/postgres -type f | sort) | sha256sum | cut -c 1-32 > ./migrations/postgres.sha256sum

import "embed"

//go:embed migrations/**/*.sql
var EmbedMigrations embed.FS
