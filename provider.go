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

	if _, err := fs.Stat(option.filesystem, dir); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("%s directory does not exist", dir)
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
	var migrations Migrations

	// Collect all SQL migration files.
	sqlMigrationFiles, err := fs.Glob(fsys, path.Join(dir, "*.sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(sqlMigrationFiles)
	for _, fileName := range sqlMigrationFiles {
		version, err := parseNumericComponent(fileName)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, &Migration{
			Source:   fileName,
			Version:  version,
			Next:     -1,
			Previous: -1,
		})
	}
	return migrations, nil
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
