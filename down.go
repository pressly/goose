package goose

import (
	"context"
	"fmt"

	"github.com/pressly/goose/v4/internal/sqlparser"
)

func (p *Provider) down(ctx context.Context, downByOne bool, version int64) ([]*MigrationResult, error) {
	if version < 0 {
		return nil, fmt.Errorf("version must be a number greater than or equal zero: %d", version)
	}

	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// feat(mf): this is where a session level advisory lock would be acquired to ensure that only
	// one goose process is running at a time. Also need to lock the Provider itself with a mutex.
	// https://github.com/pressly/goose/issues/335

	if err := p.ensureVersionTable(ctx, conn); err != nil {
		return nil, err
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
		return nil, ErrNoCurrentVersion
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
