package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"sync"

	"github.com/pressly/goose/v3/database"
)

// Provider is a goose migration goose.
type Provider struct {
	// mu protects all accesses to the provider and must be held when calling operations on the
	// database.
	mu sync.Mutex

	db    *sql.DB
	fsys  fs.FS
	cfg   config
	store database.Store

	// migrations are ordered by version in ascending order.
	migrations []*Migration
}

// NewProvider returns a new goose goose.
//
// The caller is responsible for matching the database dialect with the database/sql driver. For
// example, if the database dialect is "postgres", the database/sql driver could be
// github.com/lib/pq or github.com/jackc/pgx. Each dialect has a corresponding [database.Dialect]
// constant backed by a default [database.Store] implementation. For more advanced use cases, such
// as using a custom table name or supplying a custom store implementation, see [WithStore].
//
// fsys is the filesystem used to read the migration files, but may be nil. Most users will want to
// use [os.DirFS], os.DirFS("path/to/migrations"), to read migrations from the local filesystem.
// However, it is possible to use a different "filesystem", such as [embed.FS] or filter out
// migrations using [fs.Sub].
//
// See [ProviderOption] for more information on configuring the goose.
//
// Unless otherwise specified, all methods on Provider are safe for concurrent use.
//
// Experimental: This API is experimental and may change in the future.
func NewProvider(dialect database.Dialect, db *sql.DB, fsys fs.FS, opts ...ProviderOption) (*Provider, error) {
	if db == nil {
		return nil, errors.New("db must not be nil")
	}
	if fsys == nil {
		fsys = noopFS{}
	}
	cfg := config{
		registered:      make(map[int64]*Migration),
		excludePaths:    make(map[string]bool),
		excludeVersions: make(map[int64]bool),
	}
	for _, opt := range opts {
		if err := opt.apply(&cfg); err != nil {
			return nil, err
		}
	}
	// Allow users to specify a custom store implementation, but only if they don't specify a
	// dialect. If they specify a dialect, we'll use the default store implementation.
	if dialect == "" && cfg.store == nil {
		return nil, errors.New("dialect must not be empty")
	}
	if dialect != "" && cfg.store != nil {
		return nil, errors.New("cannot set both dialect and custom store")
	}
	var store database.Store
	if dialect != "" {
		var err error
		store, err = database.NewStore(dialect, DefaultTablename)
		if err != nil {
			return nil, err
		}
	} else {
		store = cfg.store
	}
	if store.Tablename() == "" {
		return nil, errors.New("invalid store implementation: table name must not be empty")
	}
	return newProvider(db, store, fsys, cfg, registeredGoMigrations /* global */)
}

func newProvider(
	db *sql.DB,
	store database.Store,
	fsys fs.FS,
	cfg config,
	global map[int64]*Migration,
) (*Provider, error) {
	// Collect migrations from the filesystem and merge with registered migrations.
	//
	// Note, neither of these functions parse SQL migrations by default. SQL migrations are parsed
	// lazily.
	//
	// TODO(mf): we should expose a way to parse SQL migrations eagerly. This would allow us to
	// return an error if there are any SQL parsing errors. This adds a bit overhead to startup
	// though, so we should make it optional.
	filesystemSources, err := collectFilesystemSources(fsys, false, cfg.excludePaths, cfg.excludeVersions)
	if err != nil {
		return nil, err
	}
	versionToGoMigration := make(map[int64]*Migration)
	// Add user-registered Go migrations.
	for version, m := range cfg.registered {
		versionToGoMigration[version] = m
	}
	// Add init() functions. This is a bit ugly because we need to convert from the old Migration
	// struct to the new goMigration struct.
	for version, m := range global {
		if _, ok := versionToGoMigration[version]; ok {
			return nil, fmt.Errorf("global go migration with version %d already registered with provider", version)
		}
		versionToGoMigration[version] = m
	}
	migrations, err := merge(filesystemSources, versionToGoMigration)
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

// Status returns the status of all migrations, merging the list of migrations from the database and
// filesystem. The returned items are ordered by version, in ascending order.
func (p *Provider) Status(ctx context.Context) ([]*MigrationStatus, error) {
	return p.status(ctx)
}

// GetDBVersion returns the max version from the database, regardless of the applied order. For
// example, if migrations 1,4,2,3 were applied, this method returns 4. If no migrations have been
// applied, it returns 0.
func (p *Provider) GetDBVersion(ctx context.Context) (int64, error) {
	return p.getDBVersion(ctx)
}

// ListSources returns a list of all available migration sources the provider is aware of, sorted in
// ascending order by version.
func (p *Provider) ListSources() []Source {
	sources := make([]Source, 0, len(p.migrations))
	for _, m := range p.migrations {
		sources = append(sources, Source{
			Type:    m.Type,
			Path:    m.Source,
			Version: m.Version,
		})
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

// ApplyVersion applies exactly one migration by version. If there is no source for the specified
// version, this method returns [ErrVersionNotFound]. If the migration has been applied already,
// this method returns [ErrAlreadyApplied].
//
// When direction is true, the up migration is executed, and when direction is false, the down
// migration is executed.
func (p *Provider) ApplyVersion(ctx context.Context, version int64, direction bool) (*MigrationResult, error) {
	if version < 1 {
		return nil, fmt.Errorf("invalid version: must be greater than zero: %d", version)
	}
	return p.apply(ctx, version, direction)
}

// Up applies all [StatePending] migrations. If there are no new migrations to apply, this method
// returns empty list and nil error.
func (p *Provider) Up(ctx context.Context) ([]*MigrationResult, error) {
	return p.up(ctx, false, math.MaxInt64)
}

// UpByOne applies the next available migration. If there are no migrations to apply, this method
// returns [ErrNoNextVersion]. The returned list will always have exactly one migration result.
func (p *Provider) UpByOne(ctx context.Context) (*MigrationResult, error) {
	res, err := p.up(ctx, true, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoNextVersion
	}
	// This should never happen. We should always have exactly one result and test for this.
	if len(res) > 1 {
		return nil, fmt.Errorf("unexpected number of migrations returned running up-by-one: %d", len(res))
	}
	return res[0], nil
}

// UpTo applies all available migrations up to, and including, the specified version. If there are
// no migrations to apply, this method returns empty list and nil error.
//
// For instance, if there are three new migrations (9,10,11) and the current database version is 8
// with a requested version of 10, only versions 9,10 will be applied.
func (p *Provider) UpTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	if version < 1 {
		return nil, fmt.Errorf("invalid version: must be greater than zero: %d", version)
	}
	return p.up(ctx, false, version)
}

// Down rolls back the most recently applied migration. If there are no migrations to apply, this
// method returns [ErrNoNextVersion].
func (p *Provider) Down(ctx context.Context) (*MigrationResult, error) {
	res, err := p.down(ctx, true, 0)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoNextVersion
	}
	if len(res) > 1 {
		return nil, fmt.Errorf("unexpected number of migrations returned running down: %d", len(res))
	}
	return res[0], nil
}

// DownTo rolls back all migrations down to, but not including, the specified version.
//
// For instance, if the current database version is 11,10,9... and the requested version is 9, only
// migrations 11, 10 will be rolled back.
func (p *Provider) DownTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	if version < 0 {
		return nil, fmt.Errorf("invalid version: must be a valid number or zero: %d", version)
	}
	return p.down(ctx, false, version)
}
