package goose

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Provider is a goose migration provider.
type Provider struct {
	db  *sql.DB
	opt *ProviderOptions
}

// NewProvider returns a new goose Provider.
//
// The caller is responsible for matching the database dialect with the database/sql driver. For
// example, if the database dialect is "postgres", the database/sql driver could be
// github.com/lib/pq or github.com/jackc/pgx.
//
// If opts is nil, the default options are used. See ProviderOptions for more information.
//
// Unless otherwise specified, all methods on Provider are safe for concurrent use.
func NewProvider(dialect Dialect, db *sql.DB, opts *ProviderOptions) (*Provider, error) {
	if db == nil {
		return nil, errors.New("db must not be nil")
	}
	if dialect == "" {
		return nil, errors.New("dialect must not be empty")
	}
	if opts == nil {
		opts = DefaultOptions()
	}
	if err := validateOptions(opts); err != nil {
		return nil, err
	}
	//
	// TODO(mf): implement the rest of this function
	// - db / dialect store
	// - collect sources
	// - merge sources into migrations
	return &Provider{
		db:  db,
		opt: opts,
	}, nil
}

// MigrationStatus represents the status of a single migration.
type MigrationStatus struct {
	Applied   bool
	AppliedAt time.Time
	Source    Source
}

// StatusOptions represents options for the Status method.
type StatusOptions struct{}

// Status returns the status of all migrations. The returned slice is ordered by ascending version.
//
// If opts is nil, the default options are used. See StatusOptions for more information.
func (p *Provider) Status(ctx context.Context, opts *StatusOptions) ([]*MigrationStatus, error) {
	return nil, errors.New("not implemented")
}

// GetDBVersion returns the max version from the database, regardless of when it was applied. If no
// migrations have been applied, it returns 0.
func (p *Provider) GetDBVersion(ctx context.Context) (int64, error) {
	return -1, errors.New("not implemented")
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
// the specified version, this method returns ErrNoCurrentVersion. If the migration has been applied
// already, this method returns ErrAlreadyApplied.
//
// If direction is true, the "up" migration is applied. If direction is false, the "down" migration
// is applied.
func (p *Provider) ApplyVersion(ctx context.Context, version int64, direction bool) (*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// Up applies all new migrations. If there are no new migrations to apply, this method returns empty
// list and nil error.
func (p *Provider) Up(ctx context.Context) ([]*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// UpByOne applies the next available migration. If there are no migrations to apply, this method
// returns ErrNoMigrations.
func (p *Provider) UpByOne(ctx context.Context) (*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// UpTo applies all available migrations up to and including the specified version. If there are no
// migrations to apply, this method returns empty list and nil error.
//
// For example, suppose there are 3 new migrations available 9,10,11. The current database version
// is 8 and the requested version is 10. In this scenario only versions 9,10 will be applied.
func (p *Provider) UpTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// Down rolls back the most recently applied migration. If there are no migrations to apply, this
// method returns ErrNoMigrations.
func (p *Provider) Down(ctx context.Context) (*MigrationResult, error) {
	return nil, errors.New("not implemented")
}

// DownTo rolls back all migrations down to but not including the specified version.
//
// For example, suppose the current database version is 11, and the requested version is 9. In this
// scenario only migrations 11 and 10 will be rolled back.
func (p *Provider) DownTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return nil, errors.New("not implemented")
}
