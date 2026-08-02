package goose

import (
	"fmt"
	"path/filepath"
	"time"
)

// MigrationType is the type of migration.
type MigrationType string

const (
	TypeGo  MigrationType = "go"
	TypeSQL MigrationType = "sql"
)

// Source represents a single migration source.
//
// The Path field may be empty if the migration was registered manually. This is typically the case
// for Go migrations registered using the [WithGoMigration] option.
type Source struct {
	Type    MigrationType
	Path    string
	Version int64
}

// MigrationResult is the result of a single migration operation.
type MigrationResult struct {
	Source    *Source
	Duration  time.Duration
	Direction string
	// Empty indicates no action was taken during the migration, but it was still versioned. For
	// SQL, it means no statements; for Go, it's a nil function.
	Empty bool
	// Error is only set if the migration failed.
	Error error
}

// String returns a string representation of the migration result.
//
// Example down:
//
//	EMPTY down 00006_posts_view-copy.sql (607.83µs)
//	OK    down 00005_posts_view.sql (646.25µs)
//
// Example up:
//
//	OK    up 00005_posts_view.sql (727.5µs)
//	EMPTY up 00006_posts_view-copy.sql (378.33µs)
func (m *MigrationResult) String() string {
	var format string
	if m.Direction == "up" {
		format = "%-5s %-2s %s (%s)"
	} else {
		format = "%-5s %-4s %s (%s)"
	}
	var state string
	if m.Empty {
		state = "EMPTY"
	} else {
		state = "OK"
	}
	return fmt.Sprintf(format,
		state,
		m.Direction,
		filepath.Base(m.Source.Path),
		truncateDuration(m.Duration),
	)
}

// State represents the state of a migration.
type State string

const (
	// StatePending is a migration that exists on the filesystem, but not in the database.
	StatePending State = "pending"
	// StateApplied is a migration that has been applied to the database and exists on the
	// filesystem.
	StateApplied State = "applied"
	// StateUntracked is a migration that is recorded as applied in the database, but has no
	// corresponding source on the filesystem (or registered Go migrations). This commonly happens
	// when an operator runs status against a production DB with a partial local checkout.
	StateUntracked State = "untracked"
)

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	Source    *Source
	State     State
	AppliedAt time.Time
}
