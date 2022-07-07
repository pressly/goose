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

	"github.com/pressly/goose/v3/internal/dialect"
	"github.com/pressly/goose/v3/internal/dialect/postgres"
	"github.com/pressly/goose/v3/internal/dialect/sqlite"
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
	if o.Filesystem == nil {
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
	if err := ensureMigrationTable(context.Background(), db, sqlDialect); err != nil {
		return nil, fmt.Errorf("failed goose table %s check: %w", tableName, err)
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
	}
	return nil, fmt.Errorf("database dialect %q not yet supported", dialect)
}

func collectMigrations(fsys fs.FS, dir string) (Migrations, error) {
	if _, err := fs.Stat(fsys, dir); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s directory does not exist", dir)
	}

	var unorderedMigrations Migrations

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
		unorderedMigrations = append(unorderedMigrations, &Migration{
			Source:   fileName,
			Version:  version,
			Next:     -1,
			Previous: -1,
		})
	}
	// Collect Go migrations registered via goose.AddMigration().
	for _, migration := range registeredGoMigrations {
		if _, err := parseVersion(migration.Source); err != nil {
			return nil, fmt.Errorf("could not parse go migration file %q: %w", migration.Source, err)
		}
		unorderedMigrations = append(unorderedMigrations, migration)
	}
	// Sanity check the directory does not contain versioned Go migrations that have
	// not been registred. This check ensures users didn't accidentally create a
	// go migration file and forgot to register the migration.
	//
	// This is almost always a user error and they forgot to call: func init() { goose.AddMigration(..) }
	if err := checkUnregisteredGoMigrations(fsys, dir); err != nil {
		return nil, err
	}
	return sortAndConnectMigrations(unorderedMigrations), nil
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

func (p *Provider) up(ctx context.Context, upByOne bool, version int64) error {
	if version < 1 {
		return fmt.Errorf("version must be a number greater than zero: %d", version)
	}
	if p.opt.NoVersioning {
		// This code path does not rely on database state to resolve which
		// migrations have already been applied. Instead we blindly apply
		// the requested migrations when user requests no versioning.
		if upByOne {
			// For non-versioned up-by-one this means applying the first
			// migration over and over.
			version = p.migrations[0].Version
		}
		return p.upToNoVersioning(ctx, version)
	}

	dbMigrations, err := p.listAllDBMigrations(ctx)
	if err != nil {
		return err
	}
	missingMigrations := findMissingMigrations(dbMigrations, p.migrations)

	// feature(mf): It is very possible someone may want to apply ONLY new migrations
	// and skip missing migrations altogether. At the moment this is not supported,
	// but leaving this comment because that's where that logic will be handled.
	if len(missingMigrations) > 0 && !p.opt.AllowMissing {
		var collected []string
		for _, m := range missingMigrations {
			output := fmt.Sprintf("version %d: %s", m.Version, m.Source)
			collected = append(collected, output)
		}
		return fmt.Errorf("error: found %d missing migrations:\n\t%s",
			len(missingMigrations), strings.Join(collected, "\n\t"))
	}
	if p.opt.AllowMissing {
		return p.upAllowMissing(ctx, upByOne, missingMigrations, dbMigrations)
	}

	var current int64
	for {
		var err error
		current, err = p.CurrentVersion(ctx)
		if err != nil {
			return err
		}
		next, err := p.migrations.Next(current)
		if err != nil {
			if errors.Is(err, ErrNoNextVersion) {
				break
			}
			return fmt.Errorf("failed to find next migration: %w", err)
		}
		if err := p.startMigration(ctx, true, next); err != nil {
			return err
		}
		if upByOne {
			return nil
		}
	}
	// At this point there are no more migrations to apply. But we need to maintain
	// the following behaviour:
	// UpByOne returns an error to signifying there are no more migrations.
	// Up and UpTo return nil
	if upByOne {
		return ErrNoNextVersion
	}
	return nil
}

func (p *Provider) upAllowMissing(
	ctx context.Context,
	upByOne bool,
	missingMigrations Migrations,
	dbMigrations Migrations,
) error {
	lookupApplied := make(map[int64]bool)
	for _, found := range dbMigrations {
		lookupApplied[found.Version] = true
	}
	// Apply all missing migrations first.
	for _, missing := range missingMigrations {
		if err := p.startMigration(ctx, true, missing); err != nil {
			return err
		}
		// Apply one migration and return early.
		if upByOne {
			return nil
		}
		// TODO(mf): do we need this check? It's a bit redundant, but we may
		// want to keep it as a safe-guard. Maybe we should instead have
		// the underlying query (if possible) return the current version as
		// part of the same transaction.
		currentVersion, err := p.CurrentVersion(ctx)
		if err != nil {
			return err
		}
		if currentVersion != missing.Version {
			return fmt.Errorf("error: missing migration:%d does not match current db version:%d",
				currentVersion, missing.Version)
		}

		lookupApplied[missing.Version] = true
	}
	// We can no longer rely on the database version_id to be sequential because
	// missing (out-of-order) migrations get applied before newer migrations.
	for _, found := range p.migrations {
		// TODO(mf): instead of relying on this lookup, consider hitting
		// the database directly?
		// Alternatively, we can skip a bunch migrations and start the cursor
		// at a version that represents 100% applied migrations. But this is
		// risky, and we should aim to keep this logic simple.
		if lookupApplied[found.Version] {
			continue
		}
		if err := p.startMigration(ctx, true, found); err != nil {
			return err
		}
		if upByOne {
			return nil
		}
	}
	// At this point there are no more migrations to apply. But we need to maintain
	// the following behaviour:
	// UpByOne returns an error to signifying there are no more migrations.
	// Up and UpTo return nil
	if upByOne {
		return ErrNoNextVersion
	}
	return nil
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

func (p *Provider) upToNoVersioning(ctx context.Context, version int64) error {
	for _, current := range p.migrations {
		if current.Version > version {
			return nil
		}
		if err := p.startMigration(ctx, true, current); err != nil {
			return err
		}
	}
	return nil
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

// registerNamedMigration adds up and down go migrations that are run within a transaction.
// TODO(mf): should these be exported to the user?
func registerNamedMigration(filename string, up GoMigration, down GoMigration) error {
	return register(filename, up, down, nil, nil)
}

// registerNamedMigrationNoTx adds up and down go migrations that are run outside a transaction.
// TODO(mf): should these be exported to the user?
func registerNamedMigrationNoTx(filename string, up GoMigrationNoTx, down GoMigrationNoTx) error {
	return register(filename, nil, nil, up, down)
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
		DownFn:     down,
		UpFn:       up,
		UpFnNoTx:   upNoTx,
		DownFnNoTx: downNoTx,
	}
	return nil
}

func CreateMigrationFile(dir string, sequential bool) error { return nil }
func FixMigrations(dir string) error                        { return nil }
