package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultTableName = "goose_db_version"
)

type Provider struct {
	db         *sql.DB
	migrations Migrations
	option     *options
}

func NewProvider(driverName, dbString, dir string, opts ...OptionsFunc) (*Provider, error) {
	option := &options{
		tableName:  defaultTableName,
		filesystem: osFS{},
	}
	for _, f := range opts {
		f(option)
	}

	migrations, err := collectMigrations(option.filesystem, dir)
	if err != nil {
		return nil, err
	}

	// collect all migrations
	// establish database connection
	// sanity check goose versions table exists

	var db *sql.DB

	return &Provider{
		db:         db,
		migrations: migrations,
		option:     option,
	}, nil
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

func (p *Provider) Up(ctx context.Context)      {}
func (p *Provider) UpByOne(ctx context.Context) {}
func (p *Provider) UpTo(ctx context.Context)    {}
func (p *Provider) Down(ctx context.Context)    {}
func (p *Provider) DownTo(ctx context.Context)  {}
func (p *Provider) Redo(ctx context.Context)    {}
func (p *Provider) Reset(ctx context.Context)   {}
func (p *Provider) Status(ctx context.Context)  {}
func (p *Provider) Version(ctx context.Context) {}

// EnsureDBVersion && GetDBVersion ??
func (p *Provider) GetCurrentVersion(ctx context.Context) {}

// AddMigration && AddNamedMigration ??
func (p *Provider) Register() {}

func CreateMigrationFile(dir string, sequential bool) {}
func FixMigrations(dir string)                        {}
