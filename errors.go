package goose

import "errors"

var (
	// ErrVersionNotFound when a migration version is not found.
	ErrVersionNotFound = errors.New("version not found")

	// ErrNoMigration when there are no migrations to apply. It is returned by (*Provider).Down and
	// (*Provider).UpByOne.
	ErrNoMigration = errors.New("no migration to apply")

	// ErrAlreadyApplied when a migration has already been applied.
	ErrAlreadyApplied = errors.New("already applied")
)
