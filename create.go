package goose

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

type tmplVars struct {
	Version   string
	CamelName string
}

// Create writes a new blank migration file.
func CreateWithTemplate(dir string, tmpl *template.Template, name, migrationType string, sequential bool) (string, error) {
	if tmpl == nil {
		return "", errors.New("must supply a valid template")
	}
	version := time.Now().Format(timestampFormat)
	if sequential {
		// Always use os filesystem here because this operation creates files
		// on disk.
		migrations, err := collectMigrations(osFS{}, dir)
		if err != nil {
			return "", err
		}
		vMigrations, err := migrations.versioned()
		if err != nil {
			return "", err
		}
		if last, err := vMigrations.Last(); err == nil {
			version = fmt.Sprintf(seqVersionTemplate, last.Version+1)
		} else {
			version = fmt.Sprintf(seqVersionTemplate, int64(1))
		}
	}
	filename := fmt.Sprintf("%v_%v.%v", version, snakeCase(name), migrationType)

	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		return "", fmt.Errorf("failed to create migration file: %w", err)
	}
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create migration file: %w", err)
	}
	defer f.Close()

	vars := tmplVars{
		Version:   version,
		CamelName: camelCase(name),
	}
	if err := tmpl.Execute(f, vars); err != nil {
		return "", fmt.Errorf("failed to execute tmpl: %w", err)
	}
	return f.Name(), nil
}

// Create writes a new blank migration file based on a pre-configured template.
func Create(dir, name, migrationType string, sequential bool) (string, error) {
	tmpl := new(template.Template)
	switch migrationType {
	case "sql":
		tmpl = sqlMigrationTemplate
	case "go":
		tmpl = goSQLMigrationTemplate
	default:
		return "", fmt.Errorf("unknown migration type: %q: must be either go or sql", migrationType)
	}
	return CreateWithTemplate(dir, tmpl, name, migrationType, sequential)
}

var sqlMigrationTemplate = template.Must(template.New("goose.sql-migration").Parse(`-- +goose Up
-- +goose StatementBegin
SELECT 'up SQL query';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
SELECT 'down SQL query';
-- +goose StatementEnd
`))

var goSQLMigrationTemplate = template.Must(template.New("goose.go-migration").Parse(`package migrations

import (
	"database/sql"
	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(up{{.CamelName}}, down{{.CamelName}})
}

func up{{.CamelName}}(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return nil
}

func down{{.CamelName}}(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
`))
