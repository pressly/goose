package provider

import (
	"context"

	"github.com/pressly/goose/v3/internal/sqlparser"
	"go.uber.org/multierr"
)

func (p *Provider) down(ctx context.Context, downByOne bool, version int64) (_ []*MigrationResult, retErr error) {
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
	if p.cfg.noVersioning {
		downMigrations := p.migrations
		if downByOne {
			last := p.migrations[len(p.migrations)-1]
			downMigrations = []*migration{last}
		}
		return p.runMigrations(ctx, conn, downMigrations, sqlparser.DirectionDown, downByOne)
	}
	dbMigrations, err := p.store.ListMigrations(ctx, conn)
	if err != nil {
		return nil, err
	}
	if len(dbMigrations) == 0 {
		return nil, errMissingZeroVersion
	}
	if dbMigrations[0].Version == 0 {
		return nil, nil
	}
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
