package goose

import "errors"

var (
	// ErrNoCurrentVersion when a migration version is not found.
	ErrNoCurrentVersion = errors.New("no current version found")

	// ErrNoNextVersion when the next migration version is not found.
	ErrNoNextVersion = errors.New("no next version found")

	// ErrAlreadyApplied when a migration has already been applied.
	ErrAlreadyApplied = errors.New("already applied")
)
