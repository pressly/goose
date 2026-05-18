package goose

import (
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GenerateFromDB reads SQL queries from the specified queries directory and generates
// type-safe Go code based on the schema information from migrations.
func GenerateFromDB(db *sql.DB, migrationsDir string, queriesDir string, verbose bool) error {
	if verbose {
		fmt.Println("goose: generating code from queries in", queriesDir)
	}

	// 1. Get schema information from migrations
	schema, err := extractSchemaFromMigrations(db, migrationsDir)
	if err != nil {
		return fmt.Errorf("failed to extract schema from migrations: %w", err)
	}

	// 2. Read queries from queries directory
	queries, err := readQueriesFromDirectory(queriesDir)
	if err != nil {
		return fmt.Errorf("failed to read queries: %w", err)
	}

	// 3. Generate code
	if err := generateGoCode(schema, queries, queriesDir); err != nil {
		return fmt.Errorf("failed to generate code: %w", err)
	}

	if verbose {
		fmt.Println("goose: code generation complete")
	}

	return nil
}

// Schema represents database schema information extracted from migrations
type Schema struct {
	Tables     []Table
	Enums      []Enum
	DataSource string
}

// Table represents a database table with its columns and constraints
type Table struct {
	Name    string
	Columns []Column
	Indexes []Index
}

// Column represents a database column with its type and constraints
type Column struct {
	Name     string
	Type     string
	Nullable bool
	Default  string
}

// Index represents a database index
type Index struct {
	Name    string
	Columns []string
	Unique  bool
}

// Enum represents a database enum type
type Enum struct {
	Name   string
	Values []string
}

// Query represents a SQL query from the queries directory
type Query struct {
	Name         string
	SQL          string
	Path         string
	ReadOnly     bool
	InputParams  []QueryParam
	OutputParams []QueryParam
}

// QueryParam represents a parameter for a query
type QueryParam struct {
	Name string
	Type string
}

// extractSchemaFromMigrations extracts schema information from migration files
func extractSchemaFromMigrations(db *sql.DB, dir string) (*Schema, error) {
	schema := &Schema{
		Tables:     []Table{},
		Enums:      []Enum{},
		DataSource: "postgres", // Default to postgres for now
	}

	// First, try to parse CREATE TABLE statements from migration files
	if err := parseCreateTableFromMigrations(dir, schema); err != nil {
		// If there's an error parsing migrations, fall back to database introspection
		// but log the error
		fmt.Printf("Warning: Error parsing migration files: %v\n", err)
		fmt.Println("Falling back to database introspection for schema extraction")
	}

	// If no tables were found in migrations or there was an error,
	// use database introspection as a fallback
	if len(schema.Tables) == 0 {
		// Get current database schema by examining tables in the database
		rows, err := db.Query(`
			SELECT table_name 
			FROM information_schema.tables 
			WHERE table_schema = 'public'
		`)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			var tableName string
			if err := rows.Scan(&tableName); err != nil {
				return nil, err
			}

			// Skip goose's own tables
			if tableName == "goose_db_version" {
				continue
			}

			// Get column information for this table
			table, err := getTableInfo(db, tableName)
			if err != nil {
				return nil, err
			}

			schema.Tables = append(schema.Tables, *table)
		}
	}

	return schema, nil
}

// parseCreateTableFromMigrations parses migration files to extract CREATE TABLE statements
func parseCreateTableFromMigrations(dir string, schema *Schema) error {
	// Map to track tables we've already processed
	tableMap := make(map[string]bool)

	// Regular expressions for CREATE TABLE statements
	createTableRe := regexp.MustCompile(`(?i)CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([a-zA-Z0-9_"]+)\s*\((.*?)\);`)

	// Walk through migration files
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-SQL files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".sql") {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Find CREATE TABLE statements
		matches := createTableRe.FindAllSubmatch(content, -1)
		for _, match := range matches {
			if len(match) < 3 {
				continue
			}

			// Extract table name and column definitions
			tableName := string(match[1])
			// Remove quotes if present
			tableName = strings.Trim(tableName, "\"")

			// Skip if we've already processed this table
			if tableMap[tableName] {
				continue
			}

			columnsDef := string(match[2])

			// Parse column definitions
			table := Table{
				Name:    tableName,
				Columns: parseColumns(columnsDef),
			}

			schema.Tables = append(schema.Tables, table)
			tableMap[tableName] = true
		}

		return nil
	})

	return err
}

// parseColumns parses column definitions from a CREATE TABLE statement
func parseColumns(columnsDef string) []Column {
	var columns []Column

	// Split by commas, but handle cases where commas are inside parentheses
	// This is a simplified approach and may not handle all edge cases
	parts := strings.Split(columnsDef, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Skip if it's a constraint or empty
		if part == "" || strings.HasPrefix(part, "CONSTRAINT") ||
			strings.HasPrefix(part, "PRIMARY KEY") ||
			strings.HasPrefix(part, "FOREIGN KEY") {
			continue
		}

		// Extract column name and type
		fields := strings.Fields(part)
		if len(fields) < 2 {
			continue
		}

		columnName := strings.Trim(fields[0], "\"")
		columnType := fields[1]

		// Check if column is nullable
		nullable := !strings.Contains(strings.ToUpper(part), "NOT NULL")

		columns = append(columns, Column{
			Name:     columnName,
			Type:     columnType,
			Nullable: nullable,
		})
	}

	return columns
}

// getTableInfo gets detailed information about a specific table
func getTableInfo(db *sql.DB, tableName string) (*Table, error) {
	table := &Table{
		Name:    tableName,
		Columns: []Column{},
		Indexes: []Index{},
	}

	// Get column information
	colRows, err := db.Query(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_name = $1
		ORDER BY ordinal_position
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer colRows.Close()

	for colRows.Next() {
		var (
			name       string
			dataType   string
			isNullable string
			defVal     sql.NullString
		)
		if err := colRows.Scan(&name, &dataType, &isNullable, &defVal); err != nil {
			return nil, err
		}

		column := Column{
			Name:     name,
			Type:     dataType,
			Nullable: isNullable == "YES",
		}
		if defVal.Valid {
			column.Default = defVal.String
		}

		table.Columns = append(table.Columns, column)
	}

	return table, nil
}

// readQueriesFromDirectory reads all SQL queries from the specified directory
func readQueriesFromDirectory(dir string) ([]Query, error) {
	var queries []Query

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Only process SQL files
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".sql") {
			return nil
		}

		// Read the file
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Determine if this is a read-only query based on the filename
		isReadOnly := strings.Contains(strings.ToLower(d.Name()), "_read")

		// Analyze SQL to determine parameters and return types
		sqlText := string(content)
		inputParams := analyzeQueryParams(sqlText)

		// Create a query object
		query := Query{
			Name:        strings.TrimSuffix(d.Name(), filepath.Ext(d.Name())),
			SQL:         sqlText,
			Path:        path,
			ReadOnly:    isReadOnly,
			InputParams: inputParams,
		}

		queries = append(queries, query)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return queries, nil
}

// analyzeQueryParams extracts parameter information from a SQL query
func analyzeQueryParams(sql string) []QueryParam {
	var params []QueryParam

	// Simple regex to find parameters like $1, $2, etc.
	re := regexp.MustCompile(`\$(\d+)`)
	matches := re.FindAllStringSubmatch(sql, -1)

	// Create a map to avoid duplicates
	paramMap := make(map[string]bool)

	for _, match := range matches {
		paramNum := match[1]
		paramName := "param" + paramNum

		if !paramMap[paramName] {
			params = append(params, QueryParam{
				Name: paramName,
				Type: "interface{}", // Default type, will be refined later if possible
			})
			paramMap[paramName] = true
		}
	}

	return params
}

// generateGoCode generates Go code from schema and queries
func generateGoCode(schema *Schema, queries []Query, outDir string) error {
	// Create models.go for data models based on the schema
	if err := generateModels(schema, outDir); err != nil {
		return err
	}

	// Create queries.go for SQL queries
	if err := generateQueries(schema, queries, outDir); err != nil {
		return err
	}

	// Create db.go for database connection code
	if err := generateDBCode(schema, outDir); err != nil {
		return err
	}

	return nil
}

// generateModels generates Go structs for database tables
func generateModels(schema *Schema, outDir string) error {
	// Create models directory if it doesn't exist
	modelsDir := filepath.Join(outDir, "models")
	if err := os.MkdirAll(modelsDir, 0755); err != nil {
		return err
	}

	// Generate models.go
	modelsFile := filepath.Join(modelsDir, "models.go")
	f, err := os.Create(modelsFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write package header
	fmt.Fprintln(f, "// Code generated by goose. DO NOT EDIT.")
	fmt.Fprintln(f, "package models")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "import (")
	fmt.Fprintln(f, "\t\"database/sql\"")
	fmt.Fprintln(f, "\t\"time\"")
	fmt.Fprintln(f, ")")
	fmt.Fprintln(f, "")

	// Generate a struct for each table
	for _, table := range schema.Tables {
		// Convert table name to a Go struct name (camel case)
		structName := toCamelCase(table.Name)

		fmt.Fprintf(f, "// %s represents the %s table\n", structName, table.Name)
		fmt.Fprintf(f, "type %s struct {\n", structName)

		// Generate struct fields for each column
		for _, col := range table.Columns {
			fieldName := toCamelCase(col.Name)
			goType := sqlTypeToGoType(col.Type, col.Nullable)

			fmt.Fprintf(f, "\t%s %s `db:\"%s\"`\n", fieldName, goType, col.Name)
		}

		fmt.Fprintln(f, "}")
		fmt.Fprintln(f, "")
	}

	return nil
}

// generateQueries generates Go code for database queries
func generateQueries(schema *Schema, queries []Query, outDir string) error {
	// Create queries directory if it doesn't exist
	queriesDir := filepath.Join(outDir, "queries")
	if err := os.MkdirAll(queriesDir, 0755); err != nil {
		return err
	}

	// Group queries by their filename (without extension)
	queryGroups := make(map[string][]Query)
	for _, q := range queries {
		baseName := strings.TrimSuffix(filepath.Base(q.Path), filepath.Ext(q.Path))
		queryGroups[baseName] = append(queryGroups[baseName], q)
	}

	// Generate a file for each query group
	for baseName, groupQueries := range queryGroups {
		fileName := filepath.Join(queriesDir, baseName+".go")
		f, err := os.Create(fileName)
		if err != nil {
			return err
		}

		// Write package header
		fmt.Fprintln(f, "// Code generated by goose. DO NOT EDIT.")
		fmt.Fprintln(f, "package queries")
		fmt.Fprintln(f, "")
		fmt.Fprintln(f, "import (")
		fmt.Fprintln(f, "\t\"context\"")

		// Import appropriate packages based on query type
		isReadOnly := false
		for _, q := range groupQueries {
			if strings.ToUpper(strings.TrimSpace(q.SQL[:6])) == "SELECT" {
				isReadOnly = true
				break
			}
		}

		// Add appropriate imports based on read-only status
		if isReadOnly {
			fmt.Fprintln(f, "\t\"database/sql\"")
		} else {
			fmt.Fprintln(f, "\t\"database/sql\"")
		}
		fmt.Fprintln(f, "\t\"github.com/pkg/errors\"")
		fmt.Fprintln(f, ")")
		fmt.Fprintln(f, "")

		// Generate query functions
		for _, _ = range groupQueries {
			// Process and generate each query function
			// This is a placeholder for actual query generation logic
		}

		f.Close()

	}
	return nil
}

// generateDBCode generates database connection code
func generateDBCode(schema *Schema, outDir string) error {
	// Create db.go
	dbFile := filepath.Join(outDir, "db.go")
	f, err := os.Create(dbFile)
	if err != nil {
		return err
	}
	defer f.Close()

	// Write package header
	fmt.Fprintln(f, "// Code generated by goose. DO NOT EDIT.")
	fmt.Fprintln(f, "package generated")
	fmt.Fprintln(f, "")
	fmt.Fprintln(f, "import (")
	fmt.Fprintln(f, "\t\"context\"")
	fmt.Fprintln(f, "\t\"database/sql\"")
	fmt.Fprintln(f, ")")
	fmt.Fprintln(f, "")

	// Write DB struct
	fmt.Fprintln(f, "// DB is a wrapper around sql.DB with query methods")
	fmt.Fprintln(f, "type DB struct {")
	fmt.Fprintln(f, "\tdb *sql.DB")
	fmt.Fprintln(f, "}")
	fmt.Fprintln(f, "")

	// Write New function
	fmt.Fprintln(f, "// New creates a new DB")
	fmt.Fprintln(f, "func New(db *sql.DB) *DB {")
	fmt.Fprintln(f, "\treturn &DB{db: db}")
	fmt.Fprintln(f, "}")
	fmt.Fprintln(f, "")

	// Write methods for transactions
	fmt.Fprintln(f, "// BeginTx starts a transaction")
	fmt.Fprintln(f, "func (d *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {")
	fmt.Fprintln(f, "\treturn d.db.BeginTx(ctx, opts)")
	fmt.Fprintln(f, "}")
	fmt.Fprintln(f, "")

	return nil
}

// Helper functions

// toCamelCase converts a snake_case string to CamelCase
func toCamelCase(s string) string {
	// Handle special case for empty string
	if s == "" {
		return ""
	}

	// Split by underscores and other non-alphanumeric characters
	var parts []string
	current := ""
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			current += string(c)
		} else {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	// If no parts were found, return the original string
	if len(parts) == 0 {
		return s
	}

	// Title case each part
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
		}
	}

	// Handle special case for identifiers starting with a number
	result := strings.Join(parts, "")
	if result[0] >= '0' && result[0] <= '9' {
		result = "N" + result
	}

	return result
}

// sqlTypeToGoType converts a SQL type to a Go type
func sqlTypeToGoType(sqlType string, nullable bool) string {
	var goType string

	// Normalize type by removing length specifications, etc.
	normalizedType := sqlType
	if idx := strings.IndexAny(normalizedType, "( "); idx > 0 {
		normalizedType = normalizedType[:idx]
	}
	normalizedType = strings.ToLower(normalizedType)

	switch normalizedType {
	case "int", "integer", "smallint", "int2", "int4":
		goType = "int32"
	case "bigint", "int8":
		goType = "int64"
	case "numeric", "decimal", "real", "float4":
		goType = "float32"
	case "double", "double precision", "float8":
		goType = "float64"
	case "boolean", "bool":
		goType = "bool"
	case "varchar", "char", "character", "text", "bpchar":
		goType = "string"
	case "timestamp", "timestamptz", "timestamp with time zone", "date":
		goType = "time.Time"
	case "bytea", "blob", "binary":
		goType = "[]byte"
	case "json", "jsonb":
		goType = "map[string]interface{}"
	case "uuid":
		goType = "string" // or uuid.UUID with proper imports
	case "inet", "cidr":
		goType = "string" // or net.IP with proper imports
	case "macaddr":
		goType = "string" // or net.HardwareAddr with proper imports
	default:
		// Check for array types
		if strings.HasPrefix(normalizedType, "_") || strings.HasSuffix(sqlType, "[]") {
			baseType := normalizedType
			if strings.HasPrefix(normalizedType, "_") {
				baseType = normalizedType[1:]
			} else if strings.HasSuffix(sqlType, "[]") {
				baseType = strings.TrimSuffix(sqlType, "[]")
			}
			goType = "[]" + sqlTypeToGoType(baseType, false)
		} else {
			goType = "interface{}"
		}
	}

	if nullable {
		switch goType {
		case "int32":
			return "sql.NullInt32"
		case "int64":
			return "sql.NullInt64"
		case "float32", "float64":
			return "sql.NullFloat64"
		case "bool":
			return "sql.NullBool"
		case "string":
			return "sql.NullString"
		case "time.Time":
			return "sql.NullTime"
		default:
			// For types that don't have a direct sql.Null equivalent
			return "*" + goType
		}
	}

	return goType
}

// isKeyword checks if a string is a Go keyword
func isKeyword(s string) bool {
	keywords := map[string]bool{
		"break":       true,
		"case":        true,
		"chan":        true,
		"const":       true,
		"continue":    true,
		"default":     true,
		"defer":       true,
		"else":        true,
		"fallthrough": true,
		"for":         true,
		"func":        true,
		"go":          true,
		"goto":        true,
		"if":          true,
		"import":      true,
		"interface":   true,
		"map":         true,
		"package":     true,
		"range":       true,
		"return":      true,
		"select":      true,
		"struct":      true,
		"switch":      true,
		"type":        true,
		"var":         true,
	}
	return keywords[s]
}

// safeName ensures a string is safe to use as a Go identifier
func safeName(s string) string {
	if isKeyword(s) {
		return s + "_"
	}
	return s
}
