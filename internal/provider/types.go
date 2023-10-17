package provider

import (
	"fmt"
	"time"
)

// Dialect is the type of database dialect.
type Dialect string

const (
	DialectClickHouse Dialect = "clickhouse"
	DialectMSSQL      Dialect = "mssql"
	DialectMySQL      Dialect = "mysql"
	DialectPostgres   Dialect = "postgres"
	DialectRedshift   Dialect = "redshift"
	DialectSQLite3    Dialect = "sqlite3"
	DialectTiDB       Dialect = "tidb"
	DialectVertica    Dialect = "vertica"
)

// MigrationType is the type of migration.
type MigrationType int

const (
	TypeGo MigrationType = iota + 1
	TypeSQL
)

func (t MigrationType) String() string {
	switch t {
	case TypeGo:
		return "go"
	case TypeSQL:
		return "sql"
	default:
		// This should never happen.
		return fmt.Sprintf("unknown (%d)", t)
	}
}

// Source represents a single migration source.
//
// For SQL migrations, Fullpath will always be set. For Go migrations, Fullpath will will be set if
// the migration has a corresponding file on disk. It will be empty if the migration was registered
// manually.
type Source struct {
	// Type is the type of migration.
	Type MigrationType
	// Full path to the migration file.
	//
	// Example: /path/to/migrations/001_create_users_table.sql
	Fullpath string
	// Version is the version of the migration.
	Version int64
}

// MigrationResult is the result of a single migration operation.
//
// Note, the caller is responsible for checking the Error field for any errors that occurred while
// running the migration. If the Error field is not nil, the migration failed.
type MigrationResult struct {
	Source    Source
	Duration  time.Duration
	Direction string
	// Empty is true if the file was valid, but no statements to apply. These are still versioned
	// migrations, but typically have no effect on the database.
	//
	// For SQL migrations, this means there was a valid .sql file but contained no statements. For
	// Go migrations, this means the function was nil.
	Empty bool

	// Error is any error that occurred while running the migration.
	Error error
}

// State represents the state of a migration.
type State string

const (
	// StatePending represents a migration that is on the filesystem, but not in the database.
	StatePending State = "pending"
	// StateApplied represents a migration that is in BOTH the database and on the filesystem.
	StateApplied State = "applied"

	// StateUntracked represents a migration that is in the database, but not on the filesystem.
	// StateUntracked State = "untracked"
)

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	// State is the state of the migration.
	State State
	// AppliedAt is the time the migration was applied. Only set if state is [StateApplied] or
	// [StateUntracked].
	AppliedAt time.Time
	// Source is the migration source. Only set if the state is [StatePending] or [StateApplied].
	Source Source
}
