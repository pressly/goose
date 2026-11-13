package goose

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	"github.com/pressly/goose/v3/internal/sqlparser"
)

// Validate checks migration files in dir without running them. It parses SQL
// migration files in both directions and verifies that Go migration files are
// registered in the binary.
func Validate(dir string) error {
	migrations, err := collectMigrationsFS(baseFS, dir, minVersion, maxVersion, registeredGoMigrations)
	if err != nil {
		return err
	}
	for _, m := range migrations {
		switch filepath.Ext(m.Source) {
		case ".sql":
			if err := validateSQLMigration(m.Source); err != nil {
				return err
			}
		case ".go":
			if !m.Registered {
				return fmt.Errorf("go migration %q must be registered and built into a custom binary (see https://github.com/pressly/goose/tree/master/examples/go-migrations)", m.Source)
			}
		}
	}
	return nil
}

func validateSQLMigration(filename string) error {
	f, err := baseFS.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file %q: %w", filename, err)
	}
	defer f.Close()
	by, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}
	_, useTxUp, err := sqlparser.ParseSQLMigration(bytes.NewReader(by), sqlparser.DirectionUp, verbose)
	if err != nil {
		return fmt.Errorf("failed to parse file %q: %w", filename, err)
	}
	_, useTxDown, err := sqlparser.ParseSQLMigration(bytes.NewReader(by), sqlparser.DirectionDown, verbose)
	if err != nil {
		return fmt.Errorf("failed to parse file %q: %w", filename, err)
	}
	// This is a sanity check to ensure that the parser is behaving as expected.
	if useTxUp != useTxDown {
		return fmt.Errorf("up and down statements in file %q must have the same transaction mode", filename)
	}
	return nil
}
