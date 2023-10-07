package goose

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"time"

	"github.com/pressly/goose/v3/internal/sqladapter"
)

// NewProvider returns a new goose Provider.
//
// The caller is responsible for matching the database dialect with the database/sql driver. For
// example, if the database dialect is "postgres", the database/sql driver could be
// github.com/lib/pq or github.com/jackc/pgx.
//
// fsys is the filesystem used to read the migration files. Most users will want to use
// os.DirFS("path/to/migrations") to read migrations from the local filesystem. However, it is
// possible to use a different filesystem, such as embed.FS.
//
// Functional options are used to configure the Provider. See [ProviderOption] for more information.
//
// Unless otherwise specified, all methods on Provider are safe for concurrent use.
func NewProvider(
	dialect Dialect,
	db *sql.DB,
	fsys fs.FS,
	opts ...ProviderOption,
) (*Provider, error) {
	if db == nil {
		return nil, errors.New("db must not be nil")
	}
	if dialect == "" {
		return nil, errors.New("dialect must not be empty")
	}
	if fsys == nil {
		return nil, errors.New("fsys must not be nil")
	}
	var cfg config
	for _, opt := range opts {
		if err := opt.apply(&cfg); err != nil {
			return nil, err
		}
	}
	// Set defaults
	if cfg.tableName == "" {
		cfg.tableName = defaultTablename
	}
	store, err := sqladapter.NewStore(string(dialect), cfg.tableName)
	if err != nil {
		return nil, err
	}
	// TODO(mf): implement the rest of this function - collect sources - merge sources into
	// migrations
	return &Provider{
		db:    db,
		fsys:  fsys,
		cfg:   cfg,
		store: store,
	}, nil
}

// Provider is a goose migration provider.
type Provider struct {
	db    *sql.DB
	fsys  fs.FS
	cfg   config
	store sqladapter.Store
}

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	// State represents the state of the migration. One of "untracked", "pending", "applied".
	//  - untracked: in the database, but not on the filesystem.
	//  - pending: on the filesystem, but not in the database.
	//  - applied: in both the database and on the filesystem.
	State string
	// AppliedAt is the time the migration was applied. Only set if state is applied or untracked.
	AppliedAt time.Time
	// Source is the migration source. Only set if the state is pending or applied.
	Source Source
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

// ListSources returns a list of all available migration sources the provider is aware of.
func (p *Provider) ListSources() []*Source {
	return nil
}

// Ping attempts to ping the database to verify a connection is available.
func (p *Provider) Ping(ctx context.Context) error {
	return errors.New("not implemented")
}

// Close closes the database connection.
func (p *Provider) Close() error {
	return errors.New("not implemented")
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
