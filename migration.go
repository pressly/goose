package goose

// Migration represents a single migration.
type Migration struct {
	// Type is the type of migration (SQL or Go).
	Type MigrationType
	// Source is the full path to the migration file.
	Source string
	// Version is the parsed version from the migration file name.
	Version int64
	// Empty is true if the migration is empty. For SQL migrations, this means the file contains no
	// SQL statements. For Go migrations, this means the migration functions for Up and Down is nil.
	Empty bool
	// UseTx is true if the migration is safe to run in a transaction.
	UseTx bool
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
