package goose

// Migration represents a single migration.
type Migration struct {
	// Type is the type of migration (SQL or Go).
	Type MigrationType
	// Source is the full path to the migration file.
	Source string
	// Version is the parsed version from the migration file name.
	Version int64
}

func (m *Migration) String() string {
	return m.Source
}

// MigrationType is the type of migration (SQL or Go).
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
