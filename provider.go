package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/sqlparser"
	"go.uber.org/multierr"
)

// Provider is a goose migration provider.
type Provider struct {
	// mu protects all accesses to the provider and must be held when calling operations on the
	// database.
	mu sync.Mutex

	db    *sql.DB
	store database.Store

	fsys fs.FS
	cfg  config

	// migrations are ordered by version in ascending order.
	migrations []*Migration
}

// NewProvider returns a new goose provider.
//
// The caller is responsible for matching the database dialect with the database/sql driver. For
// example, if the database dialect is "postgres", the database/sql driver could be
// github.com/lib/pq or github.com/jackc/pgx. Each dialect has a corresponding [database.Dialect]
// constant backed by a default [database.Store] implementation. For more advanced use cases, such
// as using a custom table name or supplying a custom store implementation, see [WithStore].
//
// fsys is the filesystem used to read migration files, but may be nil. Most users will want to use
// [os.DirFS], os.DirFS("path/to/migrations"), to read migrations from the local filesystem.
// However, it is possible to use a different "filesystem", such as [embed.FS] or filter out
// migrations using [fs.Sub].
//
// See [ProviderOption] for more information on configuring the provider.
//
// Unless otherwise specified, all methods on Provider are safe for concurrent use.
//
// Experimental: This API is experimental and may change in the future.
func NewProvider(dialect Dialect, db *sql.DB, fsys fs.FS, opts ...ProviderOption) (*Provider, error) {
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
		logger:          &stdLogger{},
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
		return nil, errors.New("dialect must be empty when using a custom store implementation")
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
	// Note, we don't parse SQL migrations here. They are parsed lazily when required.

	// feat(mf): we could add a flag to parse SQL migrations eagerly. This would allow us to return
	// an error if there are any SQL parsing errors. This adds a bit overhead to startup though, so
	// we should make it optional.
	filesystemSources, err := collectFilesystemSources(fsys, false, cfg.excludePaths, cfg.excludeVersions)
	if err != nil {
		return nil, err
	}
	versionToGoMigration := make(map[int64]*Migration)
	// Add user-registered Go migrations from the provider.
	for version, m := range cfg.registered {
		versionToGoMigration[version] = m
	}
	// Return an error if the global registry is explicitly disabled, but there are registered Go
	// migrations.
	if cfg.disableGlobalRegistry {
		if len(global) > 0 {
			return nil, errors.New("global registry disabled, but provider has registered go migrations")
		}
	} else {
		for version, m := range global {
			if _, ok := versionToGoMigration[version]; ok {
				return nil, fmt.Errorf("global go migration with version %d previously registered with provider", version)
			}
			versionToGoMigration[version] = m
		}
	}
	// At this point we have all registered unique Go migrations (if any). We need to merge them
	// with SQL migrations from the filesystem.
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

// GetDBVersion returns the highest version recorded in the database, regardless of the order in
// which migrations were applied. For example, if migrations were applied out of order (1,4,2,3),
// this method returns 4. If no migrations have been applied, it returns 0.
func (p *Provider) GetDBVersion(ctx context.Context) (int64, error) {
	return p.getDBMaxVersion(ctx, nil)
}

// ListSources returns a list of all migration sources known to the provider, sorted in ascending
// order by version. The path field may be empty for manually registered migrations, such as Go
// migrations registered using the [WithGoMigrations] option.
func (p *Provider) ListSources() []*Source {
	sources := make([]*Source, 0, len(p.migrations))
	for _, m := range p.migrations {
		sources = append(sources, &Source{
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

// Close closes the database connection initially supplied to the provider.
func (p *Provider) Close() error {
	return p.db.Close()
}

// ApplyVersion applies exactly one migration for the specified version. If there is no migration
// available for the specified version, this method returns [ErrVersionNotFound]. If the migration
// has already been applied, this method returns [ErrAlreadyApplied].
//
// The direction parameter determines the migration direction: true for up migration and false for
// down migration.
func (p *Provider) ApplyVersion(ctx context.Context, version int64, direction bool) (*MigrationResult, error) {
	res, err := p.apply(ctx, version, direction)
	if err != nil {
		return nil, err
	}
	// This should never happen, we must return exactly one result.
	if len(res) != 1 {
		versions := make([]string, 0, len(res))
		for _, r := range res {
			versions = append(versions, strconv.FormatInt(r.Source.Version, 10))
		}
		return nil, fmt.Errorf(
			"unexpected number of migrations applied running apply, expecting exactly one result: %v",
			strings.Join(versions, ","),
		)
	}
	return res[0], nil
}

// Up applies all pending migrations. If there are no new migrations to apply, this method returns
// empty list and nil error.
func (p *Provider) Up(ctx context.Context) ([]*MigrationResult, error) {
	return p.up(ctx, false, math.MaxInt64)
}

// UpByOne applies the next pending migration. If there is no next migration to apply, this method
// returns [ErrNoNextVersion]. The returned list will always have exactly one migration result.
func (p *Provider) UpByOne(ctx context.Context) (*MigrationResult, error) {
	res, err := p.up(ctx, true, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoNextVersion
	}
	// This should never happen, we must return exactly one result.
	if len(res) != 1 {
		versions := make([]string, 0, len(res))
		for _, r := range res {
			versions = append(versions, strconv.FormatInt(r.Source.Version, 10))
		}
		return nil, fmt.Errorf(
			"unexpected number of migrations applied running up-by-one, expecting exactly one result: %v",
			strings.Join(versions, ","),
		)
	}
	return res[0], nil
}

// UpTo applies all pending migrations up to, and including, the specified version. If there are no
// migrations to apply, this method returns empty list and nil error.
//
// For example, if there are three new migrations (9,10,11) and the current database version is 8
// with a requested version of 10, only versions 9,10 will be applied.
func (p *Provider) UpTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return p.up(ctx, false, version)
}

// Down rolls back the most recently applied migration. If there are no migrations to rollback, this
// method returns [ErrNoNextVersion].
//
// Note, migrations are rolled back in the order they were applied. And not in the reverse order of
// the migration version. This only applies in scenarios where migrations are allowed to be applied
// out of order.
func (p *Provider) Down(ctx context.Context) (*MigrationResult, error) {
	res, err := p.down(ctx, true, 0)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoNextVersion
	}
	// This should never happen, we must return exactly one result.
	if len(res) != 1 {
		versions := make([]string, 0, len(res))
		for _, r := range res {
			versions = append(versions, strconv.FormatInt(r.Source.Version, 10))
		}
		return nil, fmt.Errorf(
			"unexpected number of migrations applied running down, expecting exactly one result: %v",
			strings.Join(versions, ","),
		)
	}
	return res[0], nil
}

// DownTo rolls back all migrations down to, but not including, the specified version.
//
// For example, if the current database version is 11,10,9... and the requested version is 9, only
// migrations 11, 10 will be rolled back.
//
// Note, migrations are rolled back in the order they were applied. And not in the reverse order of
// the migration version. This only applies in scenarios where migrations are allowed to be applied
// out of order.
func (p *Provider) DownTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	if version < 0 {
		return nil, fmt.Errorf("invalid version: must be a valid number or zero: %d", version)
	}
	return p.down(ctx, false, version)
}

// *** Internal methods ***

func (p *Provider) up(
	ctx context.Context,
	byOne bool,
	version int64,
) (_ []*MigrationResult, retErr error) {
	if version < 1 {
		return nil, errInvalidVersion
	}
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	if len(p.migrations) == 0 {
		return nil, nil
	}
	var apply []*Migration
	if p.cfg.disableVersioning {
		if byOne {
			return nil, errors.New("up-by-one not supported when versioning is disabled")
		}
		apply = p.migrations
	} else {
		// optimize(mf): Listing all migrations from the database isn't great. This is only required
		// to support the allow missing (out-of-order) feature. For users that don't use this
		// feature, we could just query the database for the current max version and then apply
		// migrations greater than that version.
		dbMigrations, err := p.store.ListMigrations(ctx, conn)
		if err != nil {
			return nil, err
		}
		if len(dbMigrations) == 0 {
			return nil, errMissingZeroVersion
		}
		apply, err = p.resolveUpMigrations(dbMigrations, version)
		if err != nil {
			return nil, err
		}
	}
	return p.runMigrations(ctx, conn, apply, sqlparser.DirectionUp, byOne)
}

func (p *Provider) down(
	ctx context.Context,
	byOne bool,
	version int64,
) (_ []*MigrationResult, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	if len(p.migrations) == 0 {
		return nil, nil
	}
	if p.cfg.disableVersioning {
		var downMigrations []*Migration
		if byOne {
			last := p.migrations[len(p.migrations)-1]
			downMigrations = []*Migration{last}
		} else {
			downMigrations = p.migrations
		}
		return p.runMigrations(ctx, conn, downMigrations, sqlparser.DirectionDown, byOne)
	}
	dbMigrations, err := p.store.ListMigrations(ctx, conn)
	if err != nil {
		return nil, err
	}
	if len(dbMigrations) == 0 {
		return nil, errMissingZeroVersion
	}
	// We never migrate the zero version down.
	if dbMigrations[0].Version == 0 {
		p.printf("no migrations to run, current version: 0")
		return nil, nil
	}
	var apply []*Migration
	for _, dbMigration := range dbMigrations {
		if dbMigration.Version <= version {
			break
		}
		m, err := p.getMigration(dbMigration.Version)
		if err != nil {
			return nil, err
		}
		apply = append(apply, m)
	}
	return p.runMigrations(ctx, conn, apply, sqlparser.DirectionDown, byOne)
}

func (p *Provider) apply(
	ctx context.Context,
	version int64,
	direction bool,
) (_ []*MigrationResult, retErr error) {
	if version < 1 {
		return nil, errInvalidVersion
	}
	m, err := p.getMigration(version)
	if err != nil {
		return nil, err
	}
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	result, err := p.store.GetMigration(ctx, conn, version)
	if err != nil && !errors.Is(err, database.ErrVersionNotFound) {
		return nil, err
	}
	// There are a few states here:
	//  1. direction is up
	//    a. migration is applied, this is an error (ErrAlreadyApplied)
	//    b. migration is not applied, apply it
	if direction && result != nil {
		return nil, fmt.Errorf("version %d: %w", version, ErrAlreadyApplied)
	}
	//  2. direction is down
	//    a. migration is applied, rollback
	//    b. migration is not applied, this is an error (ErrNotApplied)
	if !direction && result == nil {
		return nil, fmt.Errorf("version %d: %w", version, ErrNotApplied)
	}
	d := sqlparser.DirectionDown
	if direction {
		d = sqlparser.DirectionUp
	}
	return p.runMigrations(ctx, conn, []*Migration{m}, d, true)
}

func (p *Provider) status(ctx context.Context) (_ []*MigrationStatus, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize: %w", err)
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	status := make([]*MigrationStatus, 0, len(p.migrations))
	for _, m := range p.migrations {
		migrationStatus := &MigrationStatus{
			Source: &Source{
				Type:    m.Type,
				Path:    m.Source,
				Version: m.Version,
			},
			State: StatePending,
		}
		dbResult, err := p.store.GetMigration(ctx, conn, m.Version)
		if err != nil && !errors.Is(err, database.ErrVersionNotFound) {
			return nil, err
		}
		if dbResult != nil {
			migrationStatus.State = StateApplied
			migrationStatus.AppliedAt = dbResult.Timestamp
		}
		status = append(status, migrationStatus)
	}

	return status, nil
}

// getDBMaxVersion returns the highest version recorded in the database, regardless of the order in
// which migrations were applied. conn may be nil, in which case a connection is initialized.
//
// optimize(mf): we should only fetch the max version from the database, no need to fetch all
// migrations only to get the max version. This means expanding the Store interface.
func (p *Provider) getDBMaxVersion(ctx context.Context, conn *sql.Conn) (_ int64, retErr error) {
	if conn == nil {
		var cleanup func() error
		var err error
		conn, cleanup, err = p.initialize(ctx)
		if err != nil {
			return 0, err
		}
		defer func() {
			retErr = multierr.Append(retErr, cleanup())
		}()
	}
	res, err := p.store.ListMigrations(ctx, conn)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, errMissingZeroVersion
	}
	// Sort in descending order.
	sort.Slice(res, func(i, j int) bool {
		return res[i].Version > res[j].Version
	})
	// Return the highest version.
	return res[0].Version, nil
}
