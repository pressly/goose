package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pressly/goose/v4/internal/dialect"
	"github.com/pressly/goose/v4/internal/sqlparser"
)

const (
	timestampFormat = "20060102150405"
)

type Provider struct {
	db         *sql.DB
	store      dialect.Store
	opt        Options
	migrations []*migration
	// feat(mf): this is where we can store the migrations in a map instead of a slice. This will
	// speed up the lookup of migrations by version.
	//
	// versions      []int64 versionToMigration map[int64]*migration
}

func NewProvider(dbDialect Dialect, db *sql.DB, opt Options) (*Provider, error) {
	internalDialect, ok := dialectLookup[dbDialect]
	if !ok {
		supported := make([]string, 0, len(dialectLookup))
		for k := range dialectLookup {
			supported = append(supported, string(k))
		}
		return nil, fmt.Errorf("invalid database dialect, must be one of: %s",
			strings.Join(supported, ","))
	}
	if db == nil {
		return nil, errors.New("db cannot be nil")
	}
	if opt.Dir == "" {
		return nil, errors.New("dir cannot be empty")
	}
	if opt.TableName == "" {
		return nil, errors.New("table name cannot be empty")
	}
	if opt.Filesystem == nil {
		return nil, errors.New("filesystem cannot be nil")
	}
	store, err := dialect.NewStore(internalDialect, opt.TableName)
	if err != nil {
		return nil, err
	}

	// feat(mf): the provider does not need to parse all the sql files on startup, they can be lazy
	// loaded when an operation is invoked. This will speed up initialization time, but may cause
	// issues if the sql files are malformed.
	//
	// There is probably an optimization in the operation itself where we look ahead and parse only
	// the required files. That way we don't end up in a situation where we commit migrations only
	// to discover half way through that there is a SQL parsing error. This partially addresses a
	// case where a migration is applied, but the next migration fails.
	// https://github.com/pressly/goose/issues/222
	migrations, err := collectMigrations(
		registeredGoMigrations,
		opt.Filesystem,
		opt.Dir,
		opt.Debug,
		opt.ExcludeFilenames,
	)
	if err != nil {
		return nil, err
	}

	return &Provider{
		db:         db,
		store:      store,
		opt:        opt,
		migrations: migrations,
	}, nil
}

func (p *Provider) ListMigrations() []Migration {
	migrations := make([]Migration, 0, len(p.migrations))
	for _, m := range p.migrations {
		migrations = append(migrations, m.toMigration())
	}
	return migrations
}

type Dialect string

const (
	DialectPostgres   Dialect = "postgres"
	DialectMySQL      Dialect = "mysql"
	DialectSQLite3    Dialect = "sqlite3"
	DialectMSSQL      Dialect = "mssql"
	DialectRedshift   Dialect = "redshift"
	DialectTiDB       Dialect = "tidb"
	DialectClickHouse Dialect = "clickhouse"
	DialectVertica    Dialect = "vertica"
)

var dialectLookup = map[Dialect]dialect.Dialect{
	DialectPostgres:   dialect.Postgres,
	DialectMySQL:      dialect.Mysql,
	DialectSQLite3:    dialect.Sqlite3,
	DialectMSSQL:      dialect.Sqlserver,
	DialectRedshift:   dialect.Redshift,
	DialectTiDB:       dialect.Tidb,
	DialectClickHouse: dialect.Clickhouse,
	DialectVertica:    dialect.Vertica,
}

// MigrationResult is the result of a successful migration operation.
type MigrationResult struct {
	Migration Migration
	Duration  time.Duration
}

// Up applies all available migrations. If there are no migrations to apply, this method returns
// empty list and nil error.
func (p *Provider) Up(ctx context.Context) ([]*MigrationResult, error) {
	return p.up(ctx, false, math.MaxInt64)
}

// UpByOne applies the next available migration. If there are no migrations to apply, this method
// returns ErrNoNextVersion.
func (p *Provider) UpByOne(ctx context.Context) (*MigrationResult, error) {
	res, err := p.up(ctx, true, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoNextVersion
	}
	return res[0], nil
}

// UpTo applies all available migrations up to and including the specified version. If there are no
// migrations to apply, this method returns empty list and nil error.
//
// For example, suppose there are 3 new migrations available 9, 10, 11. The current database version
// is 8 and the requested version is 10. In this scenario only versions 9 and 10 will be applied.
func (p *Provider) UpTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return p.up(ctx, false, version)
}

// ApplyVersion applies exactly one migration at the specified version. If a migration cannot be
// found for the specified version, this method returns ErrNoCurrentVersion. If the migration has
// been applied already, this method returns ErrAlreadyApplied.
//
// If the direction is true, the migration will be applied. If the direction is false, the migration
// will be rolled back.
func (p *Provider) ApplyVersion(ctx context.Context, version int64, direction bool) (*MigrationResult, error) {
	m, err := p.getMigration(version)
	if err != nil {
		return nil, err
	}
	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := p.ensureVersionTable(ctx, conn); err != nil {
		return nil, err
	}

	result, err := p.store.GetMigration(ctx, conn, version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if result != nil {
		return nil, ErrAlreadyApplied
	}

	d := sqlparser.DirectionDown
	if direction {
		d = sqlparser.DirectionUp
	}
	results, err := p.runMigrations(ctx, conn, []*migration{m}, d, true)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrAlreadyApplied
	}
	return results[0], nil
}

// Down rolls back the most recently applied migration.
//
// If using out-of-order migrations, this method will roll back the most recently applied migration
// that was applied out-of-order. ???
func (p *Provider) Down(ctx context.Context) (*MigrationResult, error) {
	res, err := p.down(ctx, true, 0)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoCurrentVersion
	}
	return res[0], nil
}

// DownTo rolls back all migrations down to but not including the specified version.
//
// For example, suppose we are currently at migrations 11 and the requested version is 9. In this
// scenario only migrations 11 and 10 will be rolled back.
func (p *Provider) DownTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return p.down(ctx, false, version)
}

// GetDBVersion retrieves the current database version.
func (p *Provider) GetDBVersion(ctx context.Context) (int64, error) {
	conn, err := p.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	if err := p.ensureVersionTable(ctx, conn); err != nil {
		return 0, err
	}
	res, err := p.store.ListMigrationsConn(ctx, conn)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, nil
	}
	return res[0].Version, nil
}

type RedoResult struct {
	DownResult *MigrationResult
	UpResult   *MigrationResult
}

// Redo rolls back the most recently applied migration, then runs it again.
func (p *Provider) Redo(ctx context.Context) (*RedoResult, error) {
	// feat(mf): lock the database to prevent concurrent migrations.
	downResult, err := p.Down(ctx)
	if err != nil {
		return nil, err
	}
	upResult, err := p.UpByOne(ctx)
	if err != nil {
		return nil, err
	}
	return &RedoResult{
		DownResult: downResult,
		UpResult:   upResult,
	}, nil
}

// Reset rolls back all migrations. It is an alias for DownTo(0).
func (p *Provider) Reset(ctx context.Context) ([]*MigrationResult, error) {
	return p.DownTo(ctx, 0)
}

// Ping attempts to ping the database to verify a connection is available.
func (p *Provider) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Close closes the database connection.
func (p *Provider) Close() error {
	return p.db.Close()
}

type MigrationStatus struct {
	Applied   bool
	AppliedAt time.Time
	Migration Migration
}

type StatusOptions struct {
	// IncludeNilMigrations will include a migration status for a record in the database but not in
	// the filesystem. This is common when migration files are squashed and replaced with a single
	// migration file referencing a version that has already been applied, such as
	// 00001_squashed.go.
	includeNilMigrations bool

	// Limit limits the number of migrations returned. Default is 0, which means no limit.
	limit int
	// Sort order possible values are: ASC and DESC. Default is ASC.
	order string
}

// Status returns the status of all migrations. The returned slice is ordered by ascending version.
//
// The provided StatusOptions is optional and may be nil if defaults should be used.
func (p *Provider) Status(ctx context.Context, opts *StatusOptions) ([]*MigrationStatus, error) {
	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := p.ensureVersionTable(ctx, conn); err != nil {
		return nil, err
	}

	// TODO(mf): add support for limit and order. Also would be nice to refactor the list query to
	// support limiting the set.

	status := make([]*MigrationStatus, 0, len(p.migrations))
	for _, m := range p.migrations {
		migrationStatus := &MigrationStatus{
			Migration: m.toMigration(),
		}
		dbResult, err := p.store.GetMigration(ctx, conn, m.version)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if dbResult != nil {
			migrationStatus.Applied = true
			migrationStatus.AppliedAt = dbResult.Timestamp
		}
		status = append(status, migrationStatus)
	}

	return status, nil
}

func (p *Provider) versioned() ([]*migration, error) {
	var migrations []*migration
	// assume that the user will never have more than 19700101000000 migrations
	for _, m := range p.migrations {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, fmt.Sprintf("%d", m.version))
		if versionTime.Before(time.Unix(0, 0)) || err != nil {
			migrations = append(migrations, m)
		}
	}
	return migrations, nil
}

// timestamped gets the timestamped migrations.
func (p *Provider) timestamped() ([]*migration, error) {
	var migrations []*migration
	// assume that the user will never have more than 19700101000000 migrations
	for _, m := range p.migrations {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, fmt.Sprintf("%d", m.version))
		if err != nil {
			// probably not a timestamp
			continue
		}
		if versionTime.After(time.Unix(0, 0)) {
			migrations = append(migrations, m)
		}
	}
	return migrations, nil
}

// findMissingMigrations returns a list of migrations that are missing from the database. A missing
// migration is one that has a version less than the max version in the database.
func findMissingMigrations(
	dbMigrations []*dialect.ListMigrationsResult,
	fsMigrations []*migration,
) []int64 {
	existing := make(map[int64]bool)
	var max int64
	for _, m := range dbMigrations {
		existing[m.Version] = true
		if m.Version > max {
			max = m.Version
		}
	}
	var missing []int64
	for _, m := range fsMigrations {
		if !existing[m.version] && m.version < max {
			missing = append(missing, m.version)
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		return missing[i] < missing[j]
	})
	return missing
}

// getMigration returns the migration with the given version. If no migration is found, then
// ErrNoCurrentVersion is returned.
func (p *Provider) getMigration(version int64) (*migration, error) {
	for _, m := range p.migrations {
		if m.version == version {
			return m, nil
		}
	}
	return nil, ErrNoCurrentVersion
}

func (p *Provider) ensureVersionTable(ctx context.Context, conn *sql.Conn) (retErr error) {
	// feat(mf): this is where we can check if the version table exists instead of trying to fetch
	// from a table that may not exist. https://github.com/pressly/goose/issues/461
	res, err := p.store.GetMigration(ctx, conn, 0)
	if err == nil && res != nil {
		return nil
	}
	return p.beginTx(ctx, conn, sqlparser.DirectionUp, 0, func(tx *sql.Tx) error {
		return p.store.CreateVersionTable(ctx, tx)
	})
}

func checkUnregisteredGoMigrations(
	fsys fs.FS,
	dir string,
	registered map[int64]*migration,
	exclude map[string]bool,
) error {
	goMigrationFiles, err := fs.Glob(fsys, path.Join(dir, "*.go"))
	if err != nil {
		return err
	}
	var unregistered []string
	for _, filename := range goMigrationFiles {
		if exclude[filename] {
			continue
		}
		if strings.HasSuffix(filename, "_test.go") {
			continue // Skip Go test files.
		}
		version, err := NumericComponent(filename)
		if err != nil {
			continue
		}
		// Success, skip version because it has already been registered via AddMigration or
		// AddMigrationNoTx.
		if _, ok := registered[version]; ok {
			continue
		}
		unregistered = append(unregistered, filename)
	}
	// Success, all Go migration files have been registered.
	if len(unregistered) == 0 {
		return nil
	}

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

// NumericComponent parses the version from the migration file name.
//
// XXX_descriptivename.ext where XXX specifies the version number and ext specifies the type of
// migration, either .sql or .go.
func NumericComponent(name string) (int64, error) {
	base := filepath.Base(name)
	// TODO(mf): should we silently ignore non .sql and .go files? Potentially adding an -ignore or
	// -exlude flag
	//
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
