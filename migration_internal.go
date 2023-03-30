package goose

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"

	"github.com/pressly/goose/v4/internal/sqlparser"
)

var (
	// registeredGoMigrations is a global map of all registered Go migrations.
	registeredGoMigrations = make(map[int64]*migration)
)

type migration struct {
	version int64
	source  string

	// A migration can be either a GoMigration or a SQL migration, but not both.
	// The migrationType field is used to determine which one is set.
	//
	// Note, the migration type may be sql but *sqlMigration may be nil.
	// This is because the SQL files are parsed in either the Provider
	// constructor or at the time of starting a migration operation.
	migrationType MigrationType
	goMigration   *goMigration
	sqlMigration  *sqlMigration
}

// isEmpty returns true if the migration is a registered Go migration with
// no up/down functions, or a SQL file with no valid statements.
func (m *migration) isEmpty() bool {
	if m.migrationType == MigrationTypeSQL {
		return len(m.sqlMigration.upStatements) == 0 && len(m.sqlMigration.downStatements) == 0
	}
	if m.goMigration.useTx {
		return m.goMigration.upFn == nil && m.goMigration.downFn == nil
	}
	return m.goMigration.upFnNoTx == nil && m.goMigration.downFnNoTx == nil
}

func (m *migration) toMigration() Migration {
	return Migration{
		Type:    m.migrationType,
		Source:  m.source,
		Version: m.version,
	}
}

func (m *migration) getSQLStatements(direction sqlparser.Direction) []string {
	if direction == sqlparser.DirectionDown {
		return m.sqlMigration.downStatements
	}
	return m.sqlMigration.upStatements
}

func collectMigrations(
	registered map[int64]*migration,
	fsys fs.FS,
	dir string,
	debug bool,
	excludeFilenames []string,
) ([]*migration, error) {
	if _, err := fs.Stat(fsys, dir); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}
	exclude := make(map[string]bool, len(excludeFilenames))
	for _, v := range excludeFilenames {
		exclude[v] = true
	}
	// Sanity check the directory does not contain versioned Go migrations that have
	// not been registred. This check ensures users didn't accidentally create a
	// valid looking Go migration file and forget to register it.
	//
	// This is almost always a user error.
	if err := checkUnregisteredGoMigrations(fsys, dir, registered, exclude); err != nil {
		return nil, err
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

	sqlFiles, err := fs.Glob(fsys, path.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}
	for _, filename := range sqlFiles {
		if exclude[filename] {
			continue
		}
		version, err := NumericComponent(filename)
		if err != nil {
			return nil, err
		}
		if err := checkDuplicate(version, filename); err != nil {
			return nil, err
		}
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
		upStatements, txUp, err := sqlparser.ParseSQLMigration(
			bytes.NewReader(by),
			sqlparser.DirectionUp,
			debug,
		)
		if err != nil {
			return nil, err
		}
		downStatements, txDown, err := sqlparser.ParseSQLMigration(
			bytes.NewReader(by),
			sqlparser.DirectionDown,
			debug,
		)
		if err != nil {
			return nil, err
		}
		// This is a sanity check to ensure that the parser is behaving as expected.
		if txUp != txDown {
			return nil, fmt.Errorf("up and down statements must have the same transaction mode")
		}
		unsorted[version] = &migration{
			migrationType: MigrationTypeSQL,
			source:        filename,
			version:       version,
			sqlMigration: &sqlMigration{
				useTx:          txUp,
				upStatements:   upStatements,
				downStatements: downStatements,
			},
		}
	}

	for _, goMigration := range registered {
		if exclude[goMigration.source] {
			continue
		}
		if _, err := NumericComponent(goMigration.source); err != nil {
			return nil, err
		}
		if err := checkDuplicate(goMigration.version, goMigration.source); err != nil {
			return nil, err
		}
		unsorted[goMigration.version] = goMigration
	}

	all := make([]*migration, 0, len(unsorted))
	for _, u := range unsorted {
		all = append(all, u)
	}
	// Sort migrations in ascending order by version id
	sort.Slice(all, func(i, j int) bool {
		return all[i].version < all[j].version
	})
	return all, nil
}

type goMigration struct {
	// We use an explicit bool instead of relying on pointer because all funcs
	// may be nil, but registered. For example: goose.AddMigration(nil, nil)
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
