package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/pressly/goose/v4/internal/dialectadapter"
	"github.com/pressly/goose/v4/internal/migration"
	"github.com/pressly/goose/v4/internal/sqlparser"
)

const (
	// timestampFormat is the format used for versioned timestamped migrations.
	// For example: 20230519192509_add_users_table.sql
	timestampFormat = "20060102150405"
)

// Provider is a goose migration provider.
type Provider struct {
	mu sync.Mutex

	db     *sql.DB
	store  dialectadapter.Store
	locker Locker

	opt        Options
	migrations []*migration.Migration
}

// NewProvider creates a new goose migration provider.
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
	if err := validateMandatoryOptions(opt); err != nil {
		return nil, err
	}
	store, err := dialectadapter.NewStore(internalDialect, opt.TableName)
	if err != nil {
		return nil, err
	}

	var locker Locker

	if opt.LockMode != LockModeNone {
		if opt.CustomLocker == nil {
			switch internalDialect {
			case dialectadapter.Postgres:
				locker = NewPostgresLocker(PostgresLockerOptions{})
			default:
				return nil, ErrLockNotImplemented
			}
		} else {
			locker = opt.CustomLocker
		}
	}

	sources, err := collect(opt.Filesystem, opt.Dir, true, opt.ExcludeFilenames)
	if err != nil {
		return nil, err
	}
	// TODO(mf): we can expose a provider option to allow registering migrations via options
	migrations, err := mergeMigrations(sources, registeredGoMigrations)
	if err != nil {
		return nil, err
	}
	return &Provider{
		db:         db,
		store:      store,
		locker:     locker,
		opt:        opt,
		migrations: migrations,
	}, nil
}

func mergeMigrations(
	sources []*Source,
	registered map[int64]*migration.Migration,
) ([]*migration.Migration, error) {
	var migrations []*migration.Migration
	var unregistered []string
	// Keep track of seen versions to detect duplicates. The map value is the base filename.
	seenVersions := make(map[int64]string)
	for _, src := range sources {
		currentBase := filepath.Base(src.Fullpath)
		// Check for duplicate versions. This should never happen because the sources should have
		// already been validated.
		if existing, ok := seenVersions[src.Version]; ok {
			return nil, fmt.Errorf("found duplicate migration version %d:\n\texisting:%v\n\tcurrent:%v",
				src.Version,
				existing,
				currentBase,
			)
		}
		seenVersions[src.Version] = currentBase
		// Create a migration from the source
		m := &migration.Migration{
			Fullpath: src.Fullpath,
			Version:  src.Version,
		}
		switch ext := filepath.Ext(src.Fullpath); ext {
		case ".go":
			m.Type = migration.TypeGo
			if goMigration, ok := registered[src.Version]; ok {
				m = goMigration
			} else {
				unregistered = append(unregistered, currentBase)
			}
		case ".sql":
			m.Type = migration.TypeSQL
			// SQL migrations are lazily parsed
			m.SQLParsed = false
		default:
			return nil, fmt.Errorf("unknown migration extension %q in file %s", ext, currentBase)
		}
		if m.Go != nil && m.SQL != nil {
			return nil, fmt.Errorf("migration %q has both go and sql migrations", currentBase)
		}
		// Add the migration and insert the version into the seen map
		migrations = append(migrations, m)
	}
	if len(unregistered) > 0 {
		// Return an error if the given sources contain a versioned Go migration that has not been
		// registered. This is a sanity check to ensure users didn't accidentally create a valid
		// looking Go migration file and forget to register it.
		//
		// This is almost always a user error.
		return nil, unregisteredError(unregistered)
	}
	// Sort migrations by version in ascending order
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})
	return migrations, nil
}

func unregisteredError(unregistered []string) error {
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

func (p *Provider) ListSources() []*Source {
	sources := make([]*Source, 0, len(p.migrations))
	for _, m := range p.migrations {
		src := convertMigration(m)
		sources = append(sources, src)
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

// getMigration returns the migration with the given version. If no migration is found, then
// ErrVersionNotFound is returned.
func (p *Provider) getMigration(version int64) (*migration.Migration, error) {
	for _, m := range p.migrations {
		if m.Version == version {
			return m, nil
		}
	}
	return nil, ErrVersionNotFound
}

func (p *Provider) ensureVersionTable(ctx context.Context, conn *sql.Conn) (retErr error) {
	// feat(mf): this is where we can check if the version table exists instead of trying to fetch
	// from a table that may not exist. https://github.com/pressly/goose/issues/461
	res, err := p.store.GetMigration(ctx, conn, 0)
	if err == nil && res != nil {
		return nil
	}
	return p.beginTx(ctx, conn, func(tx *sql.Tx) error {
		if err := p.store.CreateVersionTable(ctx, tx); err != nil {
			return err
		}
		if p.opt.NoVersioning {
			return nil
		}
		return p.store.InsertOrDelete(ctx, tx, true, 0)
	})
}

func validateMandatoryOptions(opt Options) error {
	if opt.Dir == "" {
		return errors.New("dir cannot be empty")
	}
	if opt.TableName == "" {
		return errors.New("table name cannot be empty")
	}
	if opt.Filesystem == nil {
		return errors.New("filesystem cannot be nil")
	}
	return nil
}

// Up
//
//
//
//

// Up applies all available migrations. If there are no migrations to apply, this method returns
// empty list and nil error.
//
// It is safe for concurrent use.
func (p *Provider) Up(ctx context.Context) ([]*MigrationResult, error) {
	return p.up(ctx, false, math.MaxInt64)
}

// UpByOne applies the next available migration. If there are no migrations to apply, this method
// returns ErrNoMigrations.
//
// It is safe for concurrent use.
func (p *Provider) UpByOne(ctx context.Context) (*MigrationResult, error) {
	res, err := p.up(ctx, true, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoMigration
	}
	return res[0], nil
}

// UpTo applies all available migrations up to and including the specified version. If there are no
// migrations to apply, this method returns empty list and nil error.
//
// For example, suppose there are 3 new migrations available 9, 10, 11. The current database version
// is 8 and the requested version is 10. In this scenario only versions 9 and 10 will be applied.
//
// It is safe for concurrent use.
func (p *Provider) UpTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return p.up(ctx, false, version)
}

func (p *Provider) up(ctx context.Context, upByOne bool, version int64) (_ []*MigrationResult, retErr error) {
	if version < 1 {
		return nil, fmt.Errorf("version must be a number greater than zero: %d", version)
	}

	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, cleanup())
	}()

	if len(p.migrations) == 0 {
		return nil, nil
	}

	if p.opt.NoVersioning {
		return p.runMigrations(ctx, conn, p.migrations, sqlparser.DirectionUp, upByOne)
	}

	// optimize(mf): Listing all migrations from the database isn't great. This is only required to
	// support the out-of-order (allow missing) feature. For users who don't use this feature, we
	// could just query the database for the current version and then apply migrations that are
	// greater than that version.
	dbMigrations, err := p.store.ListMigrationsConn(ctx, conn)
	if err != nil {
		return nil, err
	}
	dbMaxVersion := dbMigrations[0].Version
	// lookupAppliedInDB is a map of all applied migrations in the database.
	lookupAppliedInDB := make(map[int64]bool)
	for _, m := range dbMigrations {
		lookupAppliedInDB[m.Version] = true
	}

	missingMigrations := findMissingMigrations(dbMigrations, p.migrations, dbMaxVersion)

	// feature(mf): It is very possible someone may want to apply ONLY new migrations and skip
	// missing migrations entirely. At the moment this is not supported, but leaving this comment
	// because that's where that logic will be handled.
	if len(missingMigrations) > 0 && !p.opt.AllowMissing {
		var collected []string
		for _, v := range missingMigrations {
			collected = append(collected, v.filename)
		}
		msg := "migration"
		if len(collected) > 1 {
			msg += "s"
		}
		return nil, fmt.Errorf("found %d missing (out-of-order) %s: [%s]",
			len(missingMigrations), msg, strings.Join(collected, ","))
	}

	var migrationsToApply []*migration.Migration
	if p.opt.AllowMissing {
		for _, v := range missingMigrations {
			m, err := p.getMigration(v.versionID)
			if err != nil {
				return nil, err
			}
			migrationsToApply = append(migrationsToApply, m)
		}
	}
	// filter all migrations with a version greater than the supplied version (min) and less than or
	// equal to the requested version (max).
	for _, m := range p.migrations {
		if lookupAppliedInDB[m.Version] {
			continue
		}
		if m.Version > dbMaxVersion && m.Version <= version {
			migrationsToApply = append(migrationsToApply, m)
		}
	}

	// feat(mf): this is where can (optionally) group multiple migrations to be run in a single
	// transaction. The default is to apply each migration sequentially on its own.
	// https://github.com/pressly/goose/issues/222
	//
	// Note, we can't use a single transaction for all migrations because some may have to be run in
	// their own transaction.

	return p.runMigrations(ctx, conn, migrationsToApply, sqlparser.DirectionUp, upByOne)
}

type missingMigration struct {
	versionID int64
	filename  string
}

// findMissingMigrations returns a list of migrations that are missing from the database. A missing
// migration is one that has a version less than the max version in the database.
func findMissingMigrations(
	dbMigrations []*dialectadapter.ListMigrationsResult,
	fsMigrations []*migration.Migration,
	dbMaxVersion int64,
) []missingMigration {
	existing := make(map[int64]bool)
	for _, m := range dbMigrations {
		existing[m.Version] = true
	}
	var missing []missingMigration
	for _, m := range fsMigrations {
		if !existing[m.Version] && m.Version < dbMaxVersion {
			missing = append(missing, missingMigration{
				versionID: m.Version,
				filename:  filepath.Base(m.Fullpath),
			})
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		return missing[i].versionID < missing[j].versionID
	})
	return missing
}

// Down
//
//
//
//

// Down rolls back the most recently applied migration. If there are no migrations to apply, this
// method returns ErrNoMigrations.
//
// If using out-of-order migrations, this method will roll back the most recently applied migration
// that was applied out-of-order. ???
func (p *Provider) Down(ctx context.Context) (*MigrationResult, error) {
	res, err := p.down(ctx, true, 0)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoMigration
	}
	return res[0], nil
}

// DownTo rolls back all migrations down to but not including the specified version.
//
// For example, suppose we are currently at migrations 11 and the requested version is 9. In this
// scenario only migrations 11 and 10 will be rolled back.
//
// It is safe for concurrent use.
func (p *Provider) DownTo(ctx context.Context, version int64) ([]*MigrationResult, error) {
	return p.down(ctx, false, version)
}

func (p *Provider) down(ctx context.Context, downByOne bool, version int64) (_ []*MigrationResult, retErr error) {
	if version < 0 {
		return nil, fmt.Errorf("version must be a number greater than or equal zero: %d", version)
	}

	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, cleanup())
	}()

	if len(p.migrations) == 0 {
		return nil, nil
	}

	if p.opt.NoVersioning {
		var downMigrations []*migration.Migration
		if downByOne {
			downMigrations = append(downMigrations, p.migrations[len(p.migrations)-1])
		} else {
			downMigrations = p.migrations
		}
		return p.runMigrations(ctx, conn, downMigrations, sqlparser.DirectionDown, downByOne)
	}

	dbMigrations, err := p.store.ListMigrationsConn(ctx, conn)
	if err != nil {
		return nil, err
	}
	if dbMigrations[0].Version == 0 {
		return nil, nil
	}

	var downMigrations []*migration.Migration
	for _, dbMigration := range dbMigrations {
		if dbMigration.Version <= version {
			break
		}
		m, err := p.getMigration(dbMigration.Version)
		if err != nil {
			return nil, err
		}
		downMigrations = append(downMigrations, m)
	}
	return p.runMigrations(ctx, conn, downMigrations, sqlparser.DirectionDown, downByOne)
}
