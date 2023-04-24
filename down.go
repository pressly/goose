package goose

import (
	"context"
	"fmt"

	"github.com/pressly/goose/v4/internal/sqlparser"
	"go.uber.org/multierr"
)

// Down rolls back the most recently applied migration.
//
// If using out-of-order migrations, this method will roll back the most recently applied migration
// that was applied out-of-order. ???
func (p *Provider) Down(ctx context.Context) (*MigrationResult, error) {
	res, err := p.down(ctx, true, 0)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoCurrentVersion
	}
	return res[0], nil
}

// DownTo rolls back all migrations down to but not including the specified version.
//
// For example, suppose we are currently at migrations 11 and the requested version is 9. In this
// scenario only migrations 11 and 10 will be rolled back.
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
		retErr = multierr.Append(retErr, cleanup())
	}()

	if len(p.migrations) == 0 {
		return nil, nil
	}

	if p.opt.NoVersioning {
		if downByOne && len(p.migrations) == 0 {
			return nil, ErrNoNextVersion
		}
		var downMigrations []*migration
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

	// This is the sequential path.

	var downMigrations []*migration
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
