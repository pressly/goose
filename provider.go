package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
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
	dir        string
	migrations Migrations
	opt        *options
	dialect    dialect.SQL
}

func NewProvider(driverName, dbstring, dir string, opts ...OptionsFunc) (*Provider, error) {
	// Things a provider needs to work properly:
	// 1. a *sql.DB or a connection string
	// 2. a driverName, which sets a dialect .. this can be a well-defined type?
	// 3. a directory name
	opt := &options{
		tableName:  defaultTableName,
		filesystem: osFS{},
		logger:     &stdLogger{},
	}
	for _, f := range opts {
		f(opt)
	}

	migrations, err := collectMigrations(opt.filesystem, dir)
	if err != nil {
		return nil, err
	}
	if opt.db == nil && dbstring == "" {
		return nil, errors.New("must supply one of *sql.DB or a database connection string")
	}
	if opt.db != nil && dbstring != "" {
		return nil, errors.New("cannot supply both *sql.DB and a database connection string")
	}
	if opt.db == nil {
		opt.db, err = sql.Open(driverName, dbstring)
		if err != nil {
			return nil, fmt.Errorf("%s: failed to open db connection", err)
		}
	}
	dialect, err := newDialect(opt.tableName, driverName)
	if err != nil {
		return nil, err
	}
	if err := ensureMigrationTable(context.Background(), opt.db, dialect); err != nil {
		return nil, fmt.Errorf("failed goose table %s check: %w", tableName, err)
	}
	return &Provider{
		dir:        dir,
		migrations: migrations,
		opt:        opt,
		dialect:    dialect,
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

func newDialect(tableName, driverName string) (dialect.SQL, error) {
	switch driverName {
	case "pgx", "postgres":
		return postgres.New(tableName)
	case "sqlite":
		return sqlite.New(tableName)
	default:
		return nil, fmt.Errorf("driver not supported: %s", driverName)
	}
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
		version, err := parseNumericComponent(fileName)
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
		if _, err := parseNumericComponent(migration.Source); err != nil {
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
		version, err := parseNumericComponent(fileName)
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

func parseNumericComponent(name string) (int64, error) {
	base := filepath.Base(name)
	// https://github.com/pressly/goose/issues/331#issuecomment-1101556360
	// Should we silently ignore non .sql and .go files ?
	// Should we add -ignore or -exclude flags?
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

func (p *Provider) Up(ctx context.Context) error {
	return p.up(ctx, false, maxVersion)
}

func (p *Provider) UpByOne(ctx context.Context) error {
	return p.up(ctx, true, maxVersion)
}

func (p *Provider) UpTo(ctx context.Context, version int64) error {
	return p.up(ctx, false, version)
}

func (p *Provider) up(ctx context.Context, upByOne bool, version int64) error {
	if version < 1 {
		return fmt.Errorf("version must be a number greater than zero: %d", version)
	}
	if p.opt.noVersioning {
		// This code path does not rely on database state to resolve which
		// migrations have already been applied. Instead be blindly applying
		// the requested migrations when user requests no versioning.
		if upByOne {
			// For non-versioned up-by-one this means keep re-applying the first
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

	fmt.Println(len(dbMigrations))
	fmt.Println(len(p.migrations))
	fmt.Println(len(missingMigrations))

	// feature(mf): It is very possible someone may want to apply ONLY new migrations
	// and skip missing migrations altogether. At the moment this is not supported,
	// but leaving this comment because that's where that logic will be handled.
	if len(missingMigrations) > 0 && !p.opt.allowMissing {
		var collected []string
		for _, m := range missingMigrations {
			output := fmt.Sprintf("version %d: %s", m.Version, m.Source)
			collected = append(collected, output)
		}
		return fmt.Errorf("error: found %d missing migrations:\n\t%s",
			len(missingMigrations), strings.Join(collected, "\n\t"))
	}
	if p.opt.allowMissing {

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

// listAllDBMigrations returns a list of migrations ordered by version id ASC.
// Note, the Migration object only has the version field set.
func (p *Provider) listAllDBMigrations(ctx context.Context) (Migrations, error) {
	rows, err := p.opt.db.QueryContext(ctx, p.dialect.ListMigrations())
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

func (p *Provider) startMigration(ctx context.Context, direction bool, m *Migration) error {
	switch filepath.Ext(m.Source) {
	case ".sql":
		f, err := p.opt.filesystem.Open(m.Source)
		if err != nil {
			return fmt.Errorf("failed to open SQL migration file: %v: %w", filepath.Base(m.Source), err)
		}
		defer f.Close()

		statements, useTx, err := parseSQLMigration(f, direction)
		if err != nil {
			return fmt.Errorf("failed to parse SQL migration file: %v: %w", filepath.Base(m.Source), err)
		}
		if useTx {
			err = p.runTx(ctx, direction, m.Version, statements)
		} else {
			err = p.runWithoutTx(ctx, direction, m.Version, statements)
		}
		return err
	case ".go":
	}
	return nil
}

func (p *Provider) runTx(ctx context.Context, direction bool, version int64, statements []string) error {
	tx, err := p.opt.db.BeginTx(ctx, nil)
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
	// By default, we version every up and down migration. The ability to disable
	// this exists for users wishing to apply ad-hoc migrations. Useful for seeding
	// a database.
	if !p.opt.noVersioning {
		if direction {
			_, err = tx.ExecContext(ctx, p.dialect.InsertVersion(version))
		} else {
			_, err = tx.ExecContext(ctx, p.dialect.DeleteVersion(version))
		}
		if err != nil {
			if outerErr := tx.Rollback(); outerErr != nil {
				return fmt.Errorf("%v: %w", outerErr, err)
			}
			return err
		}
	}
	return tx.Commit()
}

func (p *Provider) runWithoutTx(
	ctx context.Context,
	direction bool,
	version int64,
	statements []string,
) error {
	for _, query := range statements {
		if _, err := p.opt.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	if !p.opt.noVersioning {
		var err error
		if direction {
			_, err = p.opt.db.ExecContext(ctx, p.dialect.InsertVersion(version))
		} else {
			_, err = p.opt.db.ExecContext(ctx, p.dialect.DeleteVersion(version))
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) Down(ctx context.Context) error                  { return nil }
func (p *Provider) DownTo(ctx context.Context, version int64) error { return nil }

func (p *Provider) Redo(ctx context.Context) error  { return nil }
func (p *Provider) Reset(ctx context.Context) error { return nil }

// Ahhh, this is more of a "cli" command than a library command. All it does is
// print, and chances are users would want to control this behaviour. Printing
// should be left to the user.
func (p *Provider) Status(ctx context.Context) error {
	return Status(p.opt.db, p.dir)
}

// replace EnsureDBVersion && GetDBVersion ??
func (p *Provider) CurrentVersion(ctx context.Context) (int64, error) {
	var migrationRow migrationRow
	err := p.opt.db.QueryRowContext(ctx, p.dialect.GetLatestMigration()).Scan(
		&migrationRow.ID,
		&migrationRow.VersionID,
		&migrationRow.Timestamp,
	)
	if err != nil {
		return 0, err
	}
	return migrationRow.VersionID, nil
}

type GoMigrationFunc func(
	up func(tx *sql.Tx) error,
	down func(tx *sql.Tx) error,
)

// AddMigration && AddNamedMigration ?? These should probably never have been exported..
// but there are probably users abusing AddNamedMigration
func (p *Provider) Register(version int64, f GoMigrationFunc) error { return nil }

func CreateMigrationFile(dir string, sequential bool) error { return nil }
func FixMigrations(dir string) error                        { return nil }
