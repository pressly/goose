package goose

import (
	"errors"
	"fmt"
)

var (
	// ErrVersionNotFound when a migration version is not found.
	ErrVersionNotFound = errors.New("version not found")

	// ErrAlreadyApplied when a migration has already been applied.
	ErrAlreadyApplied = errors.New("already applied")

	// ErrNoMigrations is returned by [NewProvider] when no migrations are found.
	ErrNoMigrations = errors.New("no migrations found")

	// errInvalidVersion is returned when a migration version is invalid.
	errInvalidVersion = errors.New("version must be greater than 0")
)

// PartialError is returned when a migration fails, but some migrations already got applied.
type PartialError struct {
	// Applied are migrations that were applied successfully before the error occurred. May be
	// empty.
	Applied []*MigrationResult
	// Failed contains the result of the migration that failed. Cannot be nil.
	Failed *MigrationResult
	// Err is the error that occurred while running the migration and caused the failure.
	Err error
}

func (e *PartialError) Error() string {
	return fmt.Sprintf(
		"partial migration error (type:%s,version:%d): %v",
		e.Failed.Source.Type, e.Failed.Source.Version, e.Err,
	)
}
