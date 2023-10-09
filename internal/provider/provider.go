package provider

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"os"
	"time"

	"github.com/pressly/goose/v3/internal/migrate"
	"github.com/pressly/goose/v3/internal/sqladapter"
)

var (
	// ErrNoMigrations is returned by [NewProvider] when no migrations are found.
	ErrNoMigrations = errors.New("no migrations found")
)

// NewProvider returns a new goose Provider.
//
// The caller is responsible for matching the database dialect with the database/sql driver. For
// example, if the database dialect is "postgres", the database/sql driver could be
// github.com/lib/pq or github.com/jackc/pgx.
//
// fsys is the filesystem used to read the migration files, but may be nil. Most users will want to
// use os.DirFS("path/to/migrations") to read migrations from the local filesystem. However, it is
// possible to use a different filesystem, such as embed.FS or filter out migrations using fs.Sub.
//
// See [ProviderOption] for more information on configuring the provider.
//
// Unless otherwise specified, all methods on Provider are safe for concurrent use.
//
// Experimental: This API is experimental and may change in the future.
func NewProvider(dialect string, db *sql.DB, fsys fs.FS, opts ...ProviderOption) (*Provider, error) {
	if db == nil {
		return nil, errors.New("db must not be nil")
	}
	if dialect == "" {
		return nil, errors.New("dialect must not be empty")
	}
	if fsys == nil {
		fsys = noopFS{}
	}
	var cfg config
	for _, opt := range opts {
		if err := opt.apply(&cfg); err != nil {
			return nil, err
		}
	}
	// Set defaults after applying user-supplied options so option funcs can check for empty values.
	if cfg.tableName == "" {
		cfg.tableName = defaultTablename
	}
	store, err := sqladapter.NewStore(dialect, cfg.tableName)
	if err != nil {
		return nil, err
	}
	// Collect migrations from the filesystem and merge with registered migrations.
	//
	// Note, neither of these functions parse SQL migrations by default. SQL migrations are parsed
	// lazily.
	//
	// TODO(mf): we should expose a way to parse SQL migrations eagerly. This would allow us to
	// return an error if there are any SQL parsing errors. This adds a bit overhead to startup
	// though, so we should make it optional.
	sources, err := collectFileSources(fsys, false, cfg.excludes)
	if err != nil {
		return nil, err
	}
	migrations, err := merge(sources, nil)
	if err != nil {
		return nil, err
	}
	if len(migrations) == 0 {
		return nil, ErrNoMigrations
	}
	return &Provider{
		db:         db,
		fsys:       fsys,
		cfg:        cfg,
		store:      store,
		migrations: migrations,
	}, nil
}

type noopFS struct{}

var _ fs.FS = noopFS{}

func (f noopFS) Open(name string) (fs.File, error) {
	return nil, os.ErrNotExist
}

// func (f noopFS) ReadFile(name string) ([]byte, error) {
// 	return nil, os.ErrNotExist
// }

// Provider is a goose migration provider.
type Provider struct {
	db         *sql.DB
	fsys       fs.FS
	cfg        config
	store      sqladapter.Store
	migrations []*migrate.Migration
}

// State represents the state of a migration.
type State string

const (
	// StateUntracked represents a migration that is in the database, but not on the filesystem.
	StateUntracked State = "untracked"
	// StatePending represents a migration that is on the filesystem, but not in the database.
	StatePending State = "pending"
	// StateApplied represents a migration that is in BOTH the database and on the filesystem.
	StateApplied State = "applied"
)

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	// State is the state of the migration.
	State State
	// AppliedAt is the time the migration was applied. Only set if state is [StateApplied] or
	// [StateUntracked].
	AppliedAt time.Time
	// Source is the migration source. Only set if the state is [StatePending] or [StateApplied].
	Source *Source
}

// Status returns the status of all migrations, merging the list of migrations from the database and
// filesystem. The returned items are ordered by version, in ascending order.
func (p *Provider) Status(ctx context.Context) ([]*MigrationStatus, error) {
	return nil, errors.New("not implemented")
}

// GetDBVersion returns the max version from the database, regardless of the applied order. For
// example, if migrations 1,4,2,3 were applied, this method returns 4. If no migrations have been
// applied, it returns 0.
func (p *Provider) GetDBVersion(ctx context.Context) (int64, error) {
	return 0, errors.New("not implemented")
}

// SourceType represents the type of migration source.
type SourceType string

const (
	// SourceTypeSQL represents a SQL migration.
	SourceTypeSQL SourceType = "sql"
	// SourceTypeGo represents a Go migration.
	SourceTypeGo SourceType = "go"
)

// Source represents a single migration source.
//
// For SQL migrations, Fullpath will always be set. For Go migrations, Fullpath will will be set if
// the migration has a corresponding file on disk. It will be empty if the migration was registered
// manually.
type Source struct {
	// Type is the type of migration.
	Type SourceType
	// Full path to the migration file.
	//
	// Example: /path/to/migrations/001_create_users_table.sql
	Fullpath string
	// Version is the version of the migration.
	Version int64
}

// ListSources returns a list of all available migration sources the provider is aware of, sorted in
// ascending order by version.
func (p *Provider) ListSources() []*Source {
	sources := make([]*Source, 0, len(p.migrations))
	for _, m := range p.migrations {
		s := &Source{
			Fullpath: m.Fullpath,
			Version:  m.Version,
		}
		switch m.Type {
		case migrate.TypeSQL:
			s.Type = SourceTypeSQL
		case migrate.TypeGo:
			s.Type = SourceTypeGo
		}
		sources = append(sources, s)
	}
	return sources
}

// Ping attempts to ping the database to verify a connection is available.
func (p *Provider) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

// Close closes the database connection.
func (p *Provider) Close() error {
	return p.db.Close()
}

// MigrationResult represents the result of a single migration.
type MigrationResult struct{}

// ApplyVersion applies exactly one migration at the specified version. If there is no source for
// the specified version, this method returns [ErrNoCurrentVersion]. If the migration has been
// applied already, this method returns [ErrAlreadyApplied].
//
// When direction is true, the up migration is executed, and when direction is false, the down
// migration is executed.
func (p *Provider) ApplyVersion(ctx context.Context, version int64, direction bool) (*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// Up applies all pending migrations. If there are no new migrations to apply, this method returns
// empty list and nil error.
func (p *Provider) Up(ctx context.Context) ([]*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// UpByOne applies the next available migration. If there are no migrations to apply, this method
// returns [ErrNoNextVersion].
func (p *Provider) UpByOne(ctx context.Context) (*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// UpTo applies all available migrations up to and including the specified version. If there are no
// migrations to apply, this method returns empty list and nil error.
//
// For instance, if there are three new migrations (9,10,11) and the current database version is 8
// with a requested version of 10, only versions 9 and 10 will be applied.
func (p *Provider) UpTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// Down rolls back the most recently applied migration. If there are no migrations to apply, this
// method returns [ErrNoNextVersion].
func (p *Provider) Down(ctx context.Context) (*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// DownTo rolls back all migrations down to but not including the specified version.
//
// For instance, if the current database version is 11, and the requested version is 9, only
// migrations 11 and 10 will be rolled back.
func (p *Provider) DownTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return nil, errors.New("not implemented")
}
