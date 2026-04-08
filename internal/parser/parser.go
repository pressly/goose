package parser

import (
	"fmt"
	"path/filepath"
)

// ParseMigrationFiles is a wrapper function for ParseSQLMigrations
// that maintains backward compatibility
func ParseMigrationFiles(dir string) ([]Table, error) {
	absPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for migration directory: %v", err)
	}
	
	// Use the existing SQL migration parser
	tables, err := ParseSQLMigrations(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SQL migrations: %v", err)
	}
	
	return tables, nil
}
