package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Column represents a database column with name and type
type Column struct {
	Name     string
	Type     string
	Nullable bool
}

// Table represents a database table with name and columns
type Table struct {
	Name    string
	Columns []Column
}

// ParseSQLMigrations parses migration files in the given directory to extract table schemas
func ParseSQLMigrations(dir string) ([]Table, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read migration directory: %v", err)
	}

	tables := make(map[string]*Table)
	
	// Regular expressions for parsing SQL
	createTableRe := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?(\w+)\s*\(\s*(.*?)\s*\)`)
	columnDefRe := regexp.MustCompile(`(?i)(\w+)\s+(\w+)(?:\s+NOT\s+NULL)?`)
	
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			content, err := os.ReadFile(filepath.Join(dir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read migration file %s: %v", file.Name(), err)
			}
			
			// Extract CREATE TABLE statements
			matches := createTableRe.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				tableName := match[1]
				columnsText := match[2]
				
				// Check if table already exists in our map
				table, exists := tables[tableName]
				if !exists {
					table = &Table{Name: tableName}
					tables[tableName] = table
				}
				
				// Parse column definitions
				for _, columnDef := range strings.Split(columnsText, ",") {
					columnDef = strings.TrimSpace(columnDef)
					if columnDef == "" {
						continue
					}
					
					colMatches := columnDefRe.FindStringSubmatch(columnDef)
					if len(colMatches) >= 3 {
						isNullable := !strings.Contains(strings.ToUpper(columnDef), "NOT NULL")
						
						// Prevent duplicate columns
						found := false
						for _, existingCol := range table.Columns {
							if existingCol.Name == colMatches[1] {
								found = true
								break
							}
						}
						
						if !found {
							table.Columns = append(table.Columns, Column{
								Name:     colMatches[1],
								Type:     colMatches[2],
								Nullable: isNullable,
							})
						}
					}
				}
			}
		}
	}
	
	// Convert map to slice for return
	var result []Table
	for _, table := range tables {
		result = append(result, *table)
	}
	
	return result, nil
}
