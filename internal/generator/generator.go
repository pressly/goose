package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/pressly/goose/v3/internal/parser"
)

// Query represents a SQL query with name and metadata
type Query struct {
	Name       string
	SQL        string
	ReadOnly   bool
	ReturnType string
	Params     []QueryParam
}

// QueryParam represents a parameter in a SQL query
type QueryParam struct {
	Name string
	Type string
}

// TemplateColumn is used for model generation
type TemplateColumn struct {
	Name       string
	PascalName string
	GoType     string
	DbType     string
	Nullable   bool
}

// Generate processes SQL queries and generates type-safe Go code
func Generate(tables []parser.Table, queriesDir, outDir string) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	// Parse queries from directory
	queries, err := parseQueries(queriesDir, tables)
	if err != nil {
		return fmt.Errorf("failed to parse queries: %v", err)
	}

	// Generate code for each query
	for _, query := range queries {
		if err := generateQueryCode(query, outDir); err != nil {
			return fmt.Errorf("failed to generate code for query %s: %v", query.Name, err)
		}
	}

	// Generate models for tables
	if err := generateModels(tables, outDir); err != nil {
		return fmt.Errorf("failed to generate models: %v", err)
	}

	return nil
}

// parseQueries reads SQL query files and extracts metadata
func parseQueries(queriesDir string, tables []parser.Table) ([]Query, error) {
	files, err := os.ReadDir(queriesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read queries directory: %v", err)
	}

	var queries []Query

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			content, err := os.ReadFile(filepath.Join(queriesDir, file.Name()))
			if err != nil {
				return nil, fmt.Errorf("failed to read query file %s: %v", file.Name(), err)
			}

			queryName := strings.TrimSuffix(file.Name(), ".sql")
			sqlContent := string(content)
			isReadOnly := isSelectQuery(sqlContent)

			query := Query{
				Name:       queryName,
				SQL:        sqlContent,
				ReadOnly:   isReadOnly,
				ReturnType: inferReturnType(sqlContent, tables),
				Params:     inferParams(sqlContent, tables),
			}

			queries = append(queries, query)
		}
	}

	return queries, nil
}

// isSelectQuery determines if a query is read-only (SELECT)
func isSelectQuery(sql string) bool {
	return strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "SELECT")
}

// inferReturnType determines the return type for a query
func inferReturnType(sql string, tables []parser.Table) string {
	if isSelectQuery(sql) {
		// For SELECT queries, try to infer the return type from the columns
		tableName := extractTableName(sql)
		if tableName != "" {
			for _, table := range tables {
				if strings.EqualFold(table.Name, tableName) {
					return fmt.Sprintf("[]%sModel", pascalCase(table.Name))
				}
			}
		}
		return "[]map[string]interface{}"
	}

	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(sql)), "INSERT") {
		return "int64" // Return last insert ID
	}

	return "int64" // Default: number of rows affected
}

// extractTableName tries to extract the table name from a SQL query
func extractTableName(sql string) string {
	fromRegex := regexp.MustCompile(`(?i)FROM\s+(\w+)`)
	match := fromRegex.FindStringSubmatch(sql)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

// inferParams extracts parameters from a SQL query
func inferParams(sql string, tables []parser.Table) []QueryParam {
	matches := regexp.MustCompile(`\$(\d+)`).FindAllStringSubmatch(sql, -1)

	params := make([]QueryParam, 0)
	for _, match := range matches {
		paramNum := match[1]
		paramName := fmt.Sprintf("param%s", paramNum)

		// Add if not already present
		found := false
		for _, p := range params {
			if p.Name == paramName {
				found = true
				break
			}
		}

		if !found {
			params = append(params, QueryParam{
				Name: paramName,
				Type: "interface{}",
			})
		}
	}

	return params
}

// generateQueryCode generates Go code for a specific query
func generateQueryCode(query Query, outDir string) error {
	// Create a template with the subtract function
	tmpl := template.New("query")
	tmpl.Funcs(template.FuncMap{
		"subtract": func(a, b int) int {
			return a - b
		},
	})

	// Parse the template
	tmpl, err := tmpl.Parse(queryTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	outputFile := filepath.Join(outDir, snakeCase(query.Name)+".go")
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", outputFile, err)
	}
	defer f.Close()

	data := struct {
		Query      Query
		PascalName string
	}{
		Query:      query,
		PascalName: pascalCase(query.Name),
	}

	return tmpl.Execute(f, data)
}

// generateModels generates Go struct models for database tables
func generateModels(tables []parser.Table, outDir string) error {
	tmpl, err := template.New("model").Parse(modelTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	outputFile := filepath.Join(outDir, "models.go")
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", outputFile, err)
	}
	defer f.Close()

	// Convert table data for the template
	type TemplateTable struct {
		Name       string
		PascalName string
		Columns    []TemplateColumn
	}

	templateTables := make([]TemplateTable, 0, len(tables))
	for _, table := range tables {
		tt := TemplateTable{
			Name:       table.Name,
			PascalName: pascalCase(table.Name),
			Columns:    make([]TemplateColumn, 0, len(table.Columns)),
		}

		for _, col := range table.Columns {
			tt.Columns = append(tt.Columns, TemplateColumn{
				Name:       col.Name,
				PascalName: pascalCase(col.Name),
				GoType:     sqlTypeToGoType(col.Type, col.Nullable),
				DbType:     col.Type,
				Nullable:   col.Nullable,
			})
		}

		templateTables = append(templateTables, tt)
	}

	data := struct {
		Tables []TemplateTable
	}{
		Tables: templateTables,
	}

	return tmpl.Execute(f, data)
}

// sqlTypeToGoType converts a SQL type to its Go equivalent
func sqlTypeToGoType(sqlType string, nullable bool) string {
	sqlType = strings.ToLower(sqlType)

	switch {
	case strings.Contains(sqlType, "int"):
		if nullable {
			return "*int64"
		}
		return "int64"
	case strings.Contains(sqlType, "serial"):
		if nullable {
			return "*int64"
		}
		return "int64"
	case strings.Contains(sqlType, "char") || strings.Contains(sqlType, "text"):
		if nullable {
			return "*string"
		}
		return "string"
	case strings.Contains(sqlType, "bool"):
		if nullable {
			return "*bool"
		}
		return "bool"
	case strings.Contains(sqlType, "date") || strings.Contains(sqlType, "time"):
		return "time.Time"
	case strings.Contains(sqlType, "float") || strings.Contains(sqlType, "numeric") || strings.Contains(sqlType, "decimal"):
		if nullable {
			return "*float64"
		}
		return "float64"
	default:
		if nullable {
			return "interface{}"
		}
		return "string"
	}
}

// pascalCase converts a snake_case string to PascalCase
func pascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

// snakeCase ensures a string is in snake_case
func snakeCase(s string) string {
	return strings.ToLower(s)
}

// Template for query code
const queryTemplate = `// Code generated by goose generate. DO NOT EDIT.
package generated

import (
	"context"
	{{if .Query.ReadOnly}}
	"github.com/jackc/pgx/v5"
	{{else}}
	"database/sql"
	{{end}}
	"time"
)

// {{.PascalName}} provides a type-safe API for the {{.Query.Name}} query
type {{.PascalName}} struct {
	{{if .Query.ReadOnly}}
	db *pgx.Conn
	{{else}}
	db *sql.DB
	{{end}}
	query string
}

// New{{.PascalName}} creates a new instance of {{.PascalName}}
func New{{.PascalName}}({{if .Query.ReadOnly}}db *pgx.Conn{{else}}db *sql.DB{{end}}) *{{.PascalName}} {
	return &{{.PascalName}}{
		db: db,
		query: ` + "`" + `{{.Query.SQL}}` + "`" + `,
	}
}

// Execute runs the {{.Query.Name}} query with the provided parameters
{{if .Query.ReadOnly}}
func (q *{{.PascalName}}) Execute(ctx context.Context, {{range $i, $p := .Query.Params}}{{$p.Name}} {{$p.Type}}{{if lt $i (subtract (len $.Query.Params) 1)}}, {{end}}{{end}}) ({{.Query.ReturnType}}, error) {
	rows, err := q.db.Query(ctx, q.query, {{range $i, $p := .Query.Params}}{{$p.Name}}{{if lt $i (subtract (len $.Query.Params) 1)}}, {{end}}{{end}})
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results {{.Query.ReturnType}}
	for rows.Next() {
		// TODO: Implement scan logic based on return type
		var item map[string]interface{}
		if err := rows.Scan(&item); err != nil {
			return nil, err
		}
		results = append(results, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
{{else}}
func (q *{{.PascalName}}) Execute(ctx context.Context, {{range $i, $p := .Query.Params}}{{$p.Name}} {{$p.Type}}{{if lt $i (subtract (len $.Query.Params) 1)}}, {{end}}{{end}}) ({{.Query.ReturnType}}, error) {
	result, err := q.db.ExecContext(ctx, q.query, {{range $i, $p := .Query.Params}}{{$p.Name}}{{if lt $i (subtract (len $.Query.Params) 1)}}, {{end}}{{end}})
	if err != nil {
		return 0, err
	}

	{{if eq .Query.ReturnType "int64"}}
	return result.RowsAffected()
	{{else}}
	return result.LastInsertId()
	{{end}}
}
{{end}}
`

// Template for model code
const modelTemplate = `// Code generated by goose generate. DO NOT EDIT.
package generated

import (
	"time"
)

{{range .Tables}}
// {{.PascalName}}Model represents the {{.Name}} table
type {{.PascalName}}Model struct {
	{{range .Columns}}
	{{.PascalName}} {{.GoType}} ` + "`" + `db:"{{.Name}}"` + "`" + `
	{{end}}
}
{{end}}
`
