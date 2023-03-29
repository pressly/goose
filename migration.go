package goose

import (
	"fmt"
)

type Migration struct {
	// Type is the type of migration (SQL or Go).
	Type MigrationType
	// Source is the full path to the migration file.
	Source string
	// Version is the parsed version from the migration file name.
	Version int64
	// Empty is true if the migration was a no-op, but was still recorded in the database (unless no
	// versioning is enabled).
	Empty bool
	// UseTx is true if the migration was applied in a transaction.
	UseTx bool
}

func (m *Migration) String() string {
	return fmt.Sprint(m.Source)
}

type MigrationType string

const (
	MigrationTypeGo  MigrationType = "go"
	MigrationTypeSQL MigrationType = "sql"
)

func (t MigrationType) String() string {
	if t == MigrationTypeGo {
		return "Go"
	}
	return "SQL"
}
