package goose

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pressly/goose/v4/internal/sqlparser"
)

var (
	// registeredGoMigrations is a global map of all registered Go migrations.
	registeredGoMigrations = make(map[int64]*migration)
)

type migration struct {
	// source is the full path to the migration file.
	source        string
	version       int64
	migrationType MigrationType

	// A migration is either a GoMigration or a SQL migration, but never both. The migrationType
	// field is used to determine which one is set.
	//
	// Note, the sqlParsed field is used to determine if the SQL migration has been parsed. This is
	// done to avoid parsing the SQL migration if it is never needed (e.g. the user is running a Go
	// migration). Also, the majority of the time migrations are incremental, so it is likely that
	// the user will only want to run the last few migrations, and there is no need to parse ALL
	// previous migrations.
	goMigration *goMigration

	sqlParsed    bool
	sqlMigration *sqlMigration
}

// isEmpty returns true if the migration is empty. A migration is considered empty if it has no up
// or down statements.
//
// Note, for SQL migrations this must be called after the migration has been parsed.
// func (m *migration) isEmpty(direction bool) bool {
// 	switch m.migrationType {
// 	case MigrationTypeGo:
// 		if direction {
// 			return m.goMigration.upFnNoTx == nil && m.goMigration.upFn == nil
// 		}
// 		return m.goMigration.downFnNoTx == nil && m.goMigration.downFn == nil
// 	case MigrationTypeSQL:
// 		if direction {
// 			return len(m.sqlMigration.upStatements) == 0
// 		}
// 		return len(m.sqlMigration.downStatements) == 0
// 	}
// 	return false
// }

func (m *migration) useTx() bool {
	if m.migrationType == MigrationTypeSQL {
		return m.sqlMigration.useTx
	}
	return m.goMigration.useTx
}

func (m *migration) toMigration() *Migration {
	return &Migration{
		Type:    m.migrationType,
		Source:  m.source,
		Version: m.version,
	}
}

func (m *migration) getSQLStatements(direction sqlparser.Direction) ([]string, error) {
	if !m.sqlParsed || m.sqlMigration == nil {
		return nil, errors.New("SQL migration has not been parsed")
	}
	if direction == sqlparser.DirectionDown {
		return m.sqlMigration.downStatements, nil
	}
	return m.sqlMigration.upStatements, nil
}

func parseSQLMigrations(fsys fs.FS, debug bool, migrations []*migration) error {
	for _, m := range migrations {
		if m.migrationType == MigrationTypeSQL && !m.sqlParsed {
			parsedSQLMigration, err := parseSQL(fsys, m.source, debug)
			if err != nil {
				return err
			}
			m.sqlParsed = true
			m.sqlMigration = parsedSQLMigration
		}
	}
	return nil
}

func parseSQL(fsys fs.FS, filename string, debug bool) (*sqlMigration, error) {
	// We parse both up and down statements. This is done to ensure that the SQL migration is valid
	// in both directions.
	d := sqlparser.DirectionAll

	r, err := fsys.Open(filename)
	if err != nil {
		return nil, err
	}
	by, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	m := new(sqlMigration)
	var txUp, txDown bool
	if d == sqlparser.DirectionAll || d == sqlparser.DirectionUp {
		m.upStatements, txUp, err = sqlparser.ParseSQLMigration(
			bytes.NewReader(by),
			sqlparser.DirectionUp,
			debug,
		)
		if err != nil {
			return nil, err
		}
	}
	if d == sqlparser.DirectionAll || d == sqlparser.DirectionDown {
		m.downStatements, txDown, err = sqlparser.ParseSQLMigration(
			bytes.NewReader(by),
			sqlparser.DirectionDown,
			debug,
		)
		if err != nil {
			return nil, err
		}
	}
	// This is a sanity check to ensure that the parser is behaving as expected.
	if d == sqlparser.DirectionAll && txUp != txDown {
		return nil, fmt.Errorf("up and down statements must have the same transaction mode")
	}
	return m, nil
}

// collectMigrations returns a list of migrations in the given directory. The returned list is
// sorted in ascending order by version id. Note, Go migrations are gathered from the global
// registeredGoMigrations map and are gauranteed to be unique.
//
// Important, SQL migrations are not parsed, and will be lazily parsed when the migration is run.
//
// The excludeFilenames parameter is a list of filenames to exclude entirely.
func collectMigrations(fsys fs.FS, dir string, excludeFilenames []string, debug bool) ([]*migration, error) {
	if _, err := fs.Stat(fsys, dir); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}
	exclude := make(map[string]bool, len(excludeFilenames))
	for _, v := range excludeFilenames {
		exclude[v] = true
	}
	var filenames []string
	for _, pattern := range []string{"*.sql", "*.go"} {
		files, err := fs.Glob(fsys, path.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		filenames = append(filenames, files...)
	}

	unsorted := make(map[int64]*migration)

	checkDuplicate := func(version int64, filename string) error {
		existing, ok := unsorted[version]
		if ok {
			return fmt.Errorf("found duplicate migration version %d:\n\texisting:%v\n\tcurrent:%v",
				version,
				existing.source,
				filename,
			)
		}
		return nil
	}

	// Sanity check the directory does not contain versioned Go migrations that have not been
	// registred. This check ensures users didn't accidentally create a valid looking Go migration
	// file and forget to register it.
	//
	// This is almost always a user error.
	var unregistered []string

	var migrations []*migration
	for _, name := range filenames {
		base := filepath.Base(name)
		if exclude[base] {
			continue
		}
		// Skip Go test files.
		if strings.HasSuffix(base, "_test.go") {
			continue
		}
		version, err := NumericComponent(base)
		if err != nil {
			return nil, err
		}
		if err := checkDuplicate(version, name); err != nil {
			return nil, err
		}

		switch filepath.Ext(name) {
		case ".sql":
			migrations = append(migrations, &migration{
				migrationType: MigrationTypeSQL,
				source:        name,
				version:       version,
				sqlParsed:     false,
			})
		case ".go":
			if m, ok := registeredGoMigrations[version]; ok {
				// Success, version has already been registered via AddMigration or
				// AddMigrationNoTx.
				migrations = append(migrations, m)
				continue
			}
			unregistered = append(unregistered, name)
		default:
			return nil, fmt.Errorf("invalid migration file extension: %s", name)
		}
	}
	if len(unregistered) > 0 {
		return nil, unregisteredError(unregistered)
	}
	// Sort migrations in ascending order by version id
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	return migrations, nil
}

func unregisteredError(unregistered []string) error {
	f := "file"
	if len(unregistered) > 1 {
		f += "s"
	}
	var b strings.Builder

	b.WriteString(fmt.Sprintf("error: detected %d unregistered Go %s:\n", len(unregistered), f))
	for _, name := range unregistered {
		b.WriteString("\t" + name + "\n")
	}
	b.WriteString("\n")
	b.WriteString("go functions must be registered and built into a custom binary see:\nhttps://github.com/pressly/goose/tree/master/examples/go-migrations")

	return errors.New(b.String())
}

type goMigration struct {
	// We use an explicit bool instead of relying on pointer because all funcs may be nil, but
	// registered. For example: goose.AddMigration(nil, nil)
	useTx bool

	// Only one of these func pairs will be set:
	upFn, downFn GoMigration
	// -- or --
	upFnNoTx, downFnNoTx GoMigrationNoTx
}

type sqlMigration struct {
	useTx          bool
	upStatements   []string
	downStatements []string
}

// NumericComponent parses the version from the migration file name.
//
// XXX_descriptivename.ext where XXX specifies the version number and ext specifies the type of
// migration, either .sql or .go.
func NumericComponent(name string) (int64, error) {
	base := filepath.Base(name)
	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("migration file does not have .sql or .go file extension")
	}
	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no filename separator '_' found")
	}
	n, err := strconv.ParseInt(base[:idx], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse version: %w", err)
	}
	if n < 1 {
		return 0, errors.New("migration version must be greater than zero")
	}
	return n, nil
}
