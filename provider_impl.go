package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/pressly/goose/v3/internal/sqlparser"
	"go.uber.org/multierr"
)

func (p *Provider) up(
	ctx context.Context,
	upByOne bool,
	version int64,
) (_ []*MigrationResult, retErr error) {
	if version < 1 {
		return nil, errors.New("version must be greater than zero")
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
	var apply []*Migration
	if p.cfg.disableVersioning {
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
	// feat(mf): this is where can (optionally) group multiple migrations to be run in a single
	// transaction. The default is to apply each migration sequentially on its own.
	// https://github.com/pressly/goose/issues/222
	//
	// Careful, we can't use a single transaction for all migrations because some may have to be run
	// in their own transaction.
	return p.runMigrations(ctx, conn, apply, sqlparser.DirectionUp, upByOne)
}

func (p *Provider) down(
	ctx context.Context,
	downByOne bool,
	version int64,
) (_ []*MigrationResult, retErr error) {
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
	if p.cfg.disableVersioning {
		downMigrations := p.migrations
		if downByOne {
			last := p.migrations[len(p.migrations)-1]
			downMigrations = []*Migration{last}
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
	var downMigrations []*Migration
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

func (p *Provider) apply(
	ctx context.Context,
	version int64,
	direction bool,
) (_ *MigrationResult, retErr error) {
	m, err := p.getMigration(version)
	if err != nil {
		return nil, err
	}

	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	result, err := p.store.GetMigration(ctx, conn, version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	// If the migration has already been applied, return an error, unless the migration is being
	// applied in the opposite direction. In that case, we allow the migration to be applied again.
	if result != nil && direction {
		return nil, fmt.Errorf("version %d: %w", version, ErrAlreadyApplied)
	}

	d := sqlparser.DirectionDown
	if direction {
		d = sqlparser.DirectionUp
	}
	results, err := p.runMigrations(ctx, conn, []*Migration{m}, d, true)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("version %d: %w", version, ErrAlreadyApplied)
	}
	return results[0], nil
}

func (p *Provider) status(ctx context.Context) (_ []*MigrationStatus, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	// TODO(mf): add support for limit and order. Also would be nice to refactor the list query to
	// support limiting the set.

	status := make([]*MigrationStatus, 0, len(p.migrations))
	for _, m := range p.migrations {
		migrationStatus := &MigrationStatus{
			Source: Source{
				Type:    m.Type,
				Path:    m.Source,
				Version: m.Version,
			},
			State: StatePending,
		}
		dbResult, err := p.store.GetMigration(ctx, conn, m.Version)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
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

func (p *Provider) getDBVersion(ctx context.Context) (_ int64, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	res, err := p.store.ListMigrations(ctx, conn)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, nil
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Version > res[j].Version
	})
	return res[0].Version, nil
}
