package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/pressly/goose/v4/internal/dialectadapter"
	"github.com/pressly/goose/v4/internal/sqlparser"
)

const (
	timestampFormat = "20060102150405"
)

type Provider struct {
	db         *sql.DB
	store      dialectadapter.Store
	opt        Options
	migrations []*migration
}

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
	migrations, err := collectMigrations(
		opt.Filesystem,
		opt.Dir,
		opt.ExcludeFilenames,
		opt.Debug,
	)
	if err != nil {
		return nil, err
	}

	return &Provider{
		db:         db,
		store:      store,
		opt:        opt,
		migrations: migrations,
	}, nil
}

func (p *Provider) ListMigrations() []*Migration {
	migrations := make([]*Migration, 0, len(p.migrations))
	for _, m := range p.migrations {
		migrations = append(migrations, m.toMigration())
	}
	return migrations
}

// GetLastVersion returns the version of the last migration found in the migrations directory
// (sorted by version). If there are no migrations, then 0 is returned.
func (p *Provider) GetLastVersion() int64 {
	if len(p.migrations) == 0 {
		return 0
	}
	return p.migrations[len(p.migrations)-1].version
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
// ErrNoCurrentVersion is returned.
func (p *Provider) getMigration(version int64) (*migration, error) {
	for _, m := range p.migrations {
		if m.version == version {
			return m, nil
		}
	}
	return nil, ErrNoCurrentVersion
}

func (p *Provider) ensureVersionTable(ctx context.Context, conn *sql.Conn) (retErr error) {
	// feat(mf): this is where we can check if the version table exists instead of trying to fetch
	// from a table that may not exist. https://github.com/pressly/goose/issues/461
	res, err := p.store.GetMigration(ctx, conn, 0)
	if err == nil && res != nil {
		return nil
	}
	return p.beginTx(ctx, conn, sqlparser.DirectionUp, 0, func(tx *sql.Tx) error {
		return p.store.CreateVersionTable(ctx, tx)
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
