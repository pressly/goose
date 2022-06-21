package goose

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	"github.com/pkg/errors"
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
	return nil, nil
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
