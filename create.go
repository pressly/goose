package goose

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

type tmplVars struct {
	Version   string
	CamelName string
}

var (
	sequential = false

	// timeNow is used for timestamp versions so tests can freeze the clock.
	timeNow = time.Now
)

// SetSequential set whether to use sequential versioning instead of timestamp based versioning
func SetSequential(s bool) {
	sequential = s
}

const maxCreateVersionAttempts = 1000

// Create writes a new blank migration file.
func CreateWithTemplate(db *sql.DB, dir string, tmpl *template.Template, name, migrationType string) error {
	version, err := initialCreateVersion(dir)
	if err != nil {
		return err
	}

	if tmpl == nil {
		if migrationType == "go" {
			tmpl = goSQLMigrationTemplate
		} else {
			tmpl = sqlMigrationTemplate
		}
	}

	for range maxCreateVersionAttempts {
		filename := fmt.Sprintf("%v_%v.%v", version, snakeCase(name), migrationType)
		path := filepath.Join(dir, filename)

		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
		if err != nil {
			return fmt.Errorf("failed to create migration file: %w", err)
		}

		taken, checkErr := versionTakenByOtherFile(dir, version, filepath.Base(path))
		if checkErr != nil {
			f.Close()
			os.Remove(path)
			return checkErr
		}
		if taken {
			f.Close()
			if remErr := os.Remove(path); remErr != nil && !errors.Is(remErr, os.ErrNotExist) {
				return fmt.Errorf("failed to remove colliding migration file %s: %w", path, remErr)
			}
			version, err = nextCreateVersion(dir, version)
			if err != nil {
				return err
			}
			continue
		}

		vars := tmplVars{
			Version:   version,
			CamelName: camelCase(name),
		}
		if err := tmpl.Execute(f, vars); err != nil {
			f.Close()
			return fmt.Errorf("failed to execute tmpl: %w", err)
		}
		if err := f.Close(); err != nil {
			return fmt.Errorf("failed to close migration file: %w", err)
		}

		log.Printf("Created new file: %s", f.Name())
		return nil
	}

	return fmt.Errorf("failed to create migration file: could not find a unique version after %d attempts", maxCreateVersionAttempts)
}

// Create writes a new blank migration file.
func Create(db *sql.DB, dir, name, migrationType string) error {
	return CreateWithTemplate(db, dir, nil, name, migrationType)
}

func initialCreateVersion(dir string) (string, error) {
	if sequential {
		return nextSequentialVersion(dir)
	}
	return timeNow().UTC().Format(timestampFormat), nil
}

func nextCreateVersion(dir, current string) (string, error) {
	if sequential {
		return nextSequentialVersion(dir)
	}
	return nextTimestampVersion(dir, current)
}

func nextSequentialVersion(dir string) (string, error) {
	var last int64
	versions, err := existingMigrationVersions(dir)
	if err != nil {
		return "", err
	}
	for v := range versions {
		if isTimestampedVersion(v) {
			continue
		}
		if v > last {
			last = v
		}
	}
	for v := range registeredGoMigrations {
		if isTimestampedVersion(v) {
			continue
		}
		if v > last {
			last = v
		}
	}
	return fmt.Sprintf(seqVersionTemplate, last+1), nil
}

func nextTimestampVersion(dir, current string) (string, error) {
	t, err := time.Parse(timestampFormat, current)
	if err != nil {
		n, nerr := strconv.ParseInt(current, 10, 64)
		if nerr != nil {
			return "", fmt.Errorf("failed to bump migration version %q: %w", current, err)
		}
		return strconv.FormatInt(n+1, 10), nil
	}

	versions, err := existingMigrationVersions(dir)
	if err != nil {
		return "", err
	}

	t = t.Add(time.Second)
	for range maxCreateVersionAttempts {
		candidate := t.Format(timestampFormat)
		n, perr := strconv.ParseInt(candidate, 10, 64)
		if perr != nil {
			return candidate, nil
		}
		if _, exists := versions[n]; !exists {
			return candidate, nil
		}
		t = t.Add(time.Second)
	}
	return "", fmt.Errorf("failed to find a free timestamp version after %d attempts", maxCreateVersionAttempts)
}

func existingMigrationVersions(dir string) (map[int64]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s directory does not exist", dir)
		}
		return nil, err
	}
	versions := make(map[int64]string)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := e.Name()
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		v, err := NumericComponent(base)
		if err != nil {
			continue
		}
		versions[v] = base
	}
	return versions, nil
}

func versionTakenByOtherFile(dir, version, createdBase string) (bool, error) {
	n, err := strconv.ParseInt(version, 10, 64)
	if err != nil {
		// Non-numeric versions (tests that override timestampFormat) cannot collide by version.
		return false, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, err
	}
	for _, e := range entries {
		if e.IsDir() || e.Name() == createdBase {
			continue
		}
		v, err := NumericComponent(e.Name())
		if err != nil || v != n {
			continue
		}
		info, err := e.Info()
		if err != nil {
			return true, nil
		}
		// A non-empty file already finished create and owns this version.
		if info.Size() > 0 {
			return true, nil
		}
		// Concurrent in-progress creates are still empty; the lexicographically
		// first filename keeps the version so not every caller retries.
		if e.Name() < createdBase {
			return true, nil
		}
	}
	return false, nil
}

func isTimestampedVersion(v int64) bool {
	versionTime, err := time.Parse(timestampFormat, strconv.FormatInt(v, 10))
	if err != nil {
		return false
	}
	return versionTime.After(time.Unix(0, 0))
}

var sqlMigrationTemplate = template.Must(template.New("goose.sql-migration").Parse(`-- +goose Up
SELECT 'up SQL query';

-- +goose Down
SELECT 'down SQL query';
`))

var goSQLMigrationTemplate = template.Must(template.New("goose.go-migration").Parse(`package migrations

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(up{{.CamelName}}, down{{.CamelName}})
}

func up{{.CamelName}}(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return nil
}

func down{{.CamelName}}(ctx context.Context, tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
`))
