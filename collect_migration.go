package goose

import (
	"github.com/pressly/goose/v4/internal/migration"
)

var (
	// registeredGoMigrations is a global map of all registered Go migrations.
	registeredGoMigrations = make(map[int64]*migration.Migration)
)
