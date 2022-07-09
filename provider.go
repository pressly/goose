package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pressly/goose/v4/internal/dialect"
	"github.com/pressly/goose/v4/internal/dialect/mysql"
	"github.com/pressly/goose/v4/internal/dialect/postgres"
	"github.com/pressly/goose/v4/internal/dialect/sqlite"
)

var (
	ErrNoMigrations        = errors.New("no migrations")
	ErrDuplicateMigrations = errors.New("duplicate migrations")
)

const (
	defaultTableName = "goose_db_version"
)

type Provider struct {
	db         *sql.DB
	dir        string
	migrations Migrations
	opt        *Options
	dialect    dialect.SQL
	registered map[int64]*Migration
}

// RegisterNamed adds up and down go migrations that are run within a transaction.
// TODO(mf): should these be exported to the user?
func (p *Provider) RegisterNamed(filename string, up GoMigration, down GoMigration) error {
	version, err := parseVersion(filename)
	if err != nil {
		return err
	}
	if existing, ok := p.registered[version]; ok {
		return fmt.Errorf("failed to add migration %q: version %d conflicts with %q",
			filename,
			version,
			existing.Source,
		)
	}
	p.registered[version] = &Migration{
		Version:    version,
		Next:       -1,
		Previous:   -1,
		Registered: true,
		Source:     filename,
		UpFn:       up,
		DownFn:     down,
		UpFnNoTx:   nil,
		DownFnNoTx: nil,
	}
	return nil
}

// registerNamedMigrationNoTx adds up and down go migrations that are run outside a transaction.
// TODO(mf): should these be exported to the user?
func (p *Provider) RegisterNamedNoTx(filename string, up GoMigrationNoTx, down GoMigrationNoTx) error {
	return register(filename, nil, nil, up, down)
}

// Apply applies a migration at a given version up. This is only useful for testing.
func (p *Provider) Apply(ctx context.Context, version int64) error {
	migration, err := p.migrations.Current(version)
	if err != nil {
		return err
	}
	return p.startMigration(ctx, true, migration)
}

func (p *Provider) ListMigrations() Migrations {
	return p.migrations
}

type Dialect string

const (
	DialectPostgres   Dialect = "postgres"
	DialectSqlite     Dialect = "sqlite"
	DialectMySQL      Dialect = "mysql"
	DialectRedshift   Dialect = "redshift"
	DialectTiDB       Dialect = "tidb"
	DialectClickHouse Dialect = "clickhouse"
	DialectSQLServer  Dialect = "mssql"
)

var supportedDialects = []string{
	string(DialectPostgres),
	string(DialectSqlite),
	string(DialectMySQL),
	string(DialectRedshift),
	string(DialectTiDB),
	string(DialectClickHouse),
	string(DialectSQLServer),
}

type Options struct {
	// TableName is the database schema table goose records migrations.
	// Default: goose_db_version
	TableName string

	// The default is to read from the os.
	Filesystem fs.FS

	// The default is to use standard library log.
	Logger Logger

	// AllowMissing enables the ability to allow missing (out-of-order) migrations.
	//
	// Example: migrations 1,4 are applied and then version 2,3,5 are introduced.
	// If this is set to true, then goose will apply 2,3,5 instead of raising an error.
	// The final order of applied migrations will thus be: 1,4,2,3,5.
	AllowMissing bool

	// Verbose prints additional debug statements.
	Verbose bool

	// NoVersioning enables the ability to apply migrations without tracking
	// the versions in the database schema table. Useful for seeding a database.
	NoVersioning bool
}

func (o *Options) setDefaults() {
	fmt.Println("here?1")
	if o.Filesystem == nil {
		fmt.Println("here?")
		o.Filesystem = osFS{}
	}
	if o.Logger == nil {
		o.Logger = &stdLogger{}
	}
	if o.TableName == "" {
		o.TableName = defaultTableName
	}
}

// NewProvider returns a new goose provider.
//
// The database dialect determines the dialect used to construct SQL queries. It is the caller's
// responsibility to establish a database connection with the correct driver.
//
// dir is the directory from where goose migration files will be read. By default read from the
// os, but can be modified by supplying your own fs.FS interface.
//
// The Options may be nil, and sane defaults will be used.
func NewProvider(dialect Dialect, db *sql.DB, dir string, opt *Options) (*Provider, error) {
	if dialect == "" {
		return nil, fmt.Errorf("dialect cannot be empty. Must be one of: %s", strings.Join(supportedDialects, ","))
	}
	if db == nil {
		return nil, errors.New("must supply a database connection. *sql.DB cannot be nil")
	}
	if dir == "" {
		return nil, errors.New("must specify a directory containing migration files")
	}
	if opt == nil {
		opt = &Options{}
	}
	opt.setDefaults()

	sqlDialect, err := newDialect(opt.TableName, dialect)
	if err != nil {
		return nil, err
	}
	migrations, err := collectMigrations(opt.Filesystem, dir)
	if err != nil {
		return nil, err
	}
	if len(migrations) == 0 {
		return nil, fmt.Errorf("%w in directory: %s", ErrNoMigrations, dir)
	}
	if err := ensureMigrationTable(context.Background(), db, sqlDialect); err != nil {
		return nil, fmt.Errorf("failed goose table %s check: %w", defaultTableName, err)
	}
	return &Provider{
		db:         db,
		dir:        dir,
		migrations: migrations,
		opt:        opt,
		dialect:    sqlDialect,
	}, nil
}

type migrationRow struct {
	ID        int64     `db:"id"`
	VersionID int64     `db:"version_id"`
	Timestamp time.Time `db:"tstamp"`
}

func ensureMigrationTable(ctx context.Context, db *sql.DB, dialect dialect.SQL) error {
	// Because we support multiple database drivers, we cannot assert for a specific
	// table "already exists" error. This will depend on the underlying driver.
	// Instead, we attempt to fetch the initial row but invert the error check for
	// the happy path: if no error and we have a valid timestamp then we're in a valid state.
	//
	// Note, all dialects have a default timestamp, so assuming the user did not muck around
	// with the goose table, this should always be a valid (non-zero) timestamp value.
	var migrationRow migrationRow
	err := db.QueryRowContext(ctx, dialect.GetMigration(0)).Scan(
		&migrationRow.ID,
		&migrationRow.VersionID,
		&migrationRow.Timestamp,
	)
	if err == nil && !migrationRow.Timestamp.IsZero() {
		return nil
	}
	// Create table and insert the initial row with version_id = 0 in the same tx.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, dialect.CreateTable()); err != nil {
		if outerErr := tx.Rollback(); outerErr != nil {
			return fmt.Errorf("failed to create table and rollback: %w: rollback error: %v", err, outerErr)
		}
		return fmt.Errorf("failed to create table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, dialect.InsertVersion(0)); err != nil {
		if outerErr := tx.Rollback(); outerErr != nil {
			return fmt.Errorf("failed to insert initial version: %w: rollback error: %v", err, outerErr)
		}
		return fmt.Errorf("failed to insert initial version: %w", err)
	}
	return tx.Commit()
}

func newDialect(tableName string, dialect Dialect) (dialect.SQL, error) {
	switch dialect {
	case DialectPostgres:
		return postgres.New(tableName)
	case DialectSqlite:
		return sqlite.New(tableName)
	case DialectMySQL:
		return mysql.New(tableName)
	}
	return nil, fmt.Errorf("database dialect %q not yet supported", dialect)
}

func collectMigrations(fsys fs.FS, dir string) (Migrations, error) {
	if _, err := fs.Stat(fsys, dir); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}

	unsortedMigrations := make(map[int64]*Migration)

	checkDuplicate := func(version int64, filename string) error {
		existing, ok := unsortedMigrations[version]
		if ok {
			return fmt.Errorf("found %w in version %d:\n\texisting:%v\n\tcurrent:%v",
				ErrDuplicateMigrations,
				version,
				existing.Source,
				filename,
			)
		}
		return nil
	}

	// Collect all SQL migration files.
	sqlMigrationFiles, err := fs.Glob(fsys, path.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}
	for _, fileName := range sqlMigrationFiles {
		version, err := parseVersion(fileName)
		if err != nil {
			return nil, err
		}
		if err := checkDuplicate(version, fileName); err != nil {
			return nil, err
		}
		unsortedMigrations[version] = &Migration{
			Source:   fileName,
			Version:  version,
			Next:     -1,
			Previous: -1,
		}
	}
	// Collect Go migrations registered via goose.Register().
	for _, migration := range registeredGoMigrations {
		version, err := parseVersion(migration.Source)
		if err != nil {
			return nil, err
		}
		if err := checkDuplicate(version, migration.Source); err != nil {
			return nil, err
		}
		unsortedMigrations[version] = migration
	}
	// Sanity check the directory does not contain versioned Go migrations that have
	// not been registred. This check ensures users didn't accidentally create a
	// go migration file and forgot to register the migration.
	//
	// This is almost always a user error and they forgot to call: func init() { goose.Register(..) }
	if err := checkUnregisteredGoMigrations(fsys, dir); err != nil {
		return nil, err
	}
	sortedMigrations := make(Migrations, 0, len(unsortedMigrations))
	for _, migration := range unsortedMigrations {
		sortedMigrations = append(sortedMigrations, migration)
	}
	// Sort migrations in ascending order by version id
	sort.Slice(sortedMigrations, func(i, j int) bool {
		return sortedMigrations[i].Version < sortedMigrations[j].Version
	})
	// Now that we're sorted in the appropriate direction,
	// populate next and previous for each migration
	for i, m := range sortedMigrations {
		prev := int64(-1)
		if i > 0 {
			prev = sortedMigrations[i-1].Version
			sortedMigrations[i-1].Next = m.Version
		}
		sortedMigrations[i].Previous = prev
	}
	return sortedMigrations, nil
}

func checkUnregisteredGoMigrations(fsys fs.FS, dir string) error {
	goMigrationFiles, err := fs.Glob(fsys, path.Join(dir, "*.go"))
	if err != nil {
		return err
	}
	var unregisteredGoFiles []string
	for _, fileName := range goMigrationFiles {
		version, err := parseVersion(fileName)
		if err != nil {
			// TODO(mf): log warning here?
			// I think we do this because we allow _test.go files in the same
			// directory.
			continue
		}
		// Success, skip version because it has already been registered
		// via goose.AddMigration().
		if _, ok := registeredGoMigrations[version]; ok {
			continue
		}
		unregisteredGoFiles = append(unregisteredGoFiles, fileName)
	}
	// Success, all go migration files have been registered.
	if len(unregisteredGoFiles) == 0 {
		return nil
	}

	f := "file"
	if len(unregisteredGoFiles) > 1 {
		f += "s"
	}
	var b strings.Builder

	b.WriteString(fmt.Sprintf("error: detected %d unregistered go %s:\n", len(unregisteredGoFiles), f))
	for _, name := range unregisteredGoFiles {
		b.WriteString("\t" + name + "\n")
	}
	b.WriteString("\n")
	b.WriteString("go functions must be registered and built into a custom binary see:\nhttps://github.com/pressly/goose/tree/master/examples/go-migrations")

	return errors.New(b.String())
}

func parseVersion(name string) (int64, error) {
	base := filepath.Base(name)
	// TODO(mf): should we silently ignore non .sql and .go files? Potentially
	// adding an -ignore or -exlude flag
	// https://github.com/pressly/goose/issues/331#issuecomment-1101556360
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

// listAllDBMigrations returns a list of migrations ordered by version id ASC.
// Note, the Migration object only has the version field set.
func (p *Provider) listAllDBMigrations(ctx context.Context) (Migrations, error) {
	rows, err := p.db.QueryContext(ctx, p.dialect.ListMigrations())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var all Migrations
	for rows.Next() {
		var m migrationRow
		if err := rows.Scan(&m.ID, &m.VersionID, &m.Timestamp); err != nil {
			return nil, err
		}
		all = append(all, &Migration{Version: m.VersionID})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(all, func(i, j int) bool {
		return all[i].Version < all[j].Version
	})
	return all, nil
}

type EmptyGoMigrationError struct {
	Migration *Migration
}

func (e EmptyGoMigrationError) Error() string {
	return fmt.Sprintf("empty go migration: %s", filepath.Base(e.Migration.Source))
}

func (p *Provider) startMigration(ctx context.Context, direction bool, m *Migration) error {
	switch filepath.Ext(m.Source) {
	case ".sql":
		f, err := p.opt.Filesystem.Open(m.Source)
		if err != nil {
			return fmt.Errorf("failed to open SQL migration file: %v: %w", filepath.Base(m.Source), err)
		}
		defer f.Close()

		statements, useTx, err := parseSQLMigration(f, direction)
		if err != nil {
			return fmt.Errorf("failed to parse SQL migration file: %v: %w", filepath.Base(m.Source), err)
		}
		if len(statements) == 0 {
			// TODO(mf): revisit this behaviour.
			p.opt.Logger.Println("EMPTY", filepath.Base(m.Source))
			return nil
		}
		if useTx {
			return p.runTx(ctx, direction, m.Version, statements)
		}
		return p.runWithoutTx(ctx, direction, m.Version, statements)
	case ".go":
		if m.UpFnNoTx != nil || m.DownFnNoTx != nil {
			fn := m.DownFnNoTx
			if direction {
				fn = m.UpFnNoTx
			}
			// Run go migration function.
			if err := fn(p.db); err != nil {
				return fmt.Errorf("failed to run no tx go migration: %s: %w", filepath.Base(m.Source), err)
			}
			if p.opt.NoVersioning {
				return nil
			}
			return p.insertOrDeleteVersionNoTx(ctx, direction, m.Version)
		}
		// Run go-based migration within a tx.
		if m.UpFn != nil || m.DownFn != nil {
			tx, err := p.db.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to begin transaction: %w", err)
			}
			fn := m.DownFn
			if direction {
				fn = m.UpFn
			}
			// Run go migration function.
			if err := fn(tx); err != nil {
				if outerErr := tx.Rollback(); outerErr != nil {
					return fmt.Errorf("failed to run go migration: %s: %w: rollback error: %v",
						filepath.Base(m.Source),
						err,
						outerErr,
					)
				}
				return fmt.Errorf("failed to run go migration: %s: %w", filepath.Base(m.Source), err)
			}
			if !p.opt.NoVersioning {
				if err := p.insertOrDeleteVersion(ctx, tx, direction, m.Version); err != nil {
					if outerErr := tx.Rollback(); outerErr != nil {
						return fmt.Errorf("%v: %w", outerErr, err)
					}
					return err
				}
			}
			return tx.Commit()
		}
		// TODO(mf): revisit this behaviour.
		p.opt.Logger.Println("EMPTY", filepath.Base(m.Source))
		return nil
	}
	return nil
}

func (p *Provider) runTx(ctx context.Context, direction bool, version int64, statements []string) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	for _, query := range statements {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			if outerErr := tx.Rollback(); outerErr != nil {
				return fmt.Errorf("%v: %w", outerErr, err)
			}
			return err
		}
	}
	if !p.opt.NoVersioning {
		if err := p.insertOrDeleteVersion(ctx, tx, direction, version); err != nil {
			if outerErr := tx.Rollback(); outerErr != nil {
				return fmt.Errorf("%v: %w", outerErr, err)
			}
			return err
		}
	}
	return tx.Commit()
}

func (p *Provider) runWithoutTx(ctx context.Context, direction bool, version int64, statements []string) error {
	for _, query := range statements {
		if _, err := p.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	if p.opt.NoVersioning {
		return nil
	}
	return p.insertOrDeleteVersionNoTx(ctx, direction, version)
}

func (p *Provider) insertOrDeleteVersion(ctx context.Context, tx *sql.Tx, direction bool, version int64) error {
	if direction {
		_, err := tx.ExecContext(ctx, p.dialect.InsertVersion(version))
		return err
	}
	_, err := tx.ExecContext(ctx, p.dialect.DeleteVersion(version))
	return err
}

func (p *Provider) insertOrDeleteVersionNoTx(ctx context.Context, direction bool, version int64) error {
	if direction {
		_, err := p.db.ExecContext(ctx, p.dialect.InsertVersion(version))
		return err
	}
	_, err := p.db.ExecContext(ctx, p.dialect.DeleteVersion(version))
	return err
}

// GoMigration is a go migration func that is run within a transaction.
type GoMigration func(tx *sql.Tx) error

// GoMigrationNoTx is a go migration funt that is run outside a transaction.
type GoMigrationNoTx func(db *sql.DB) error

// Register adds up and down go migrations that are run within a transaction.
func Register(up GoMigration, down GoMigration) error {
	_, filename, _, _ := runtime.Caller(1)
	return register(filename, up, down, nil, nil)
}

// RegisterNoTx adds up and down go migrations that are run outside a transaction.
func RegisterNoTx(up GoMigration, down GoMigration) error {
	_, filename, _, _ := runtime.Caller(1)
	return register(filename, up, down, nil, nil)
}

func register(
	filename string,
	up GoMigration,
	down GoMigration,
	upNoTx GoMigrationNoTx,
	downNoTx GoMigrationNoTx,
) error {
	// Sanity check caller did not mix tx and non-tx based functions.
	if (up != nil || down != nil) && (upNoTx != nil || downNoTx != nil) {
		return fmt.Errorf("cannot mix tx and non-tx based go migrations functions")
	}
	version, err := parseVersion(filename)
	if err != nil {
		return err
	}
	if existing, ok := registeredGoMigrations[version]; ok {
		return fmt.Errorf("failed to add migration %q: version %d conflicts with %q",
			filename,
			version,
			existing.Source,
		)
	}
	// Add to global as a registered migration.
	registeredGoMigrations[version] = &Migration{
		Version:    version,
		Next:       -1,
		Previous:   -1,
		Registered: true,
		Source:     filename,
		UpFn:       up,
		DownFn:     down,
		UpFnNoTx:   upNoTx,
		DownFnNoTx: downNoTx,
	}
	return nil
}

func FixMigrations(dir string) error { return nil }

// findMissingMigrations migrations returns all missing migrations.
// A migrations is considered missing if it has a version less than the
// current known max version.
func findMissingMigrations(knownMigrations, newMigrations Migrations) Migrations {
	max := knownMigrations[len(knownMigrations)-1].Version
	existing := make(map[int64]bool)
	for _, known := range knownMigrations {
		existing[known.Version] = true
	}
	var missing Migrations
	for _, new := range newMigrations {
		if !existing[new.Version] && new.Version < max {
			missing = append(missing, new)
		}
	}
	sort.SliceStable(missing, func(i, j int) bool {
		return missing[i].Version < missing[j].Version
	})
	return missing
}
