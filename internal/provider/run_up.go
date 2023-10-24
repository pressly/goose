package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pressly/goose/v3/internal/sqladapter"
	"github.com/pressly/goose/v3/internal/sqlparser"
	"go.uber.org/multierr"
)

func (p *Provider) up(ctx context.Context, upByOne bool, version int64) (_ []*MigrationResult, retErr error) {
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
	var apply []*migration
	if p.cfg.noVersioning {
		apply = p.migrations
	} else {
		// optimize(mf): Listing all migrations from the database isn't great. This is only required to
		// support the allow missing (out-of-order) feature. For users that don't use this feature, we
		// could just query the database for the current max version and then apply migrations greater
		// than that version.
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

func (p *Provider) resolveUpMigrations(
	dbVersions []*sqladapter.ListMigrationsResult,
	version int64,
) ([]*migration, error) {
	var apply []*migration
	var dbMaxVersion int64
	// dbAppliedVersions is a map of all applied migrations in the database.
	dbAppliedVersions := make(map[int64]bool, len(dbVersions))
	for _, m := range dbVersions {
		dbAppliedVersions[m.Version] = true
		if m.Version > dbMaxVersion {
			dbMaxVersion = m.Version
		}
	}
	missingMigrations := findMissingMigrations(dbVersions, p.migrations)
	// feat(mf): It is very possible someone may want to apply ONLY new migrations and skip missing
	// migrations entirely. At the moment this is not supported, but leaving this comment because
	// that's where that logic would be handled.
	//
	// For example, if db has 1,4 applied and 2,3,5 are new, we would apply only 5 and skip 2,3.
	// Not sure if this is a common use case, but it's possible.
	if len(missingMigrations) > 0 && !p.cfg.allowMissing {
		var collected []string
		for _, v := range missingMigrations {
			collected = append(collected, v.filename)
		}
		msg := "migration"
		if len(collected) > 1 {
			msg += "s"
		}
		return nil, fmt.Errorf("found %d missing (out-of-order) %s lower than current max (%d): [%s]",
			len(missingMigrations), msg, dbMaxVersion, strings.Join(collected, ","),
		)
	}
	for _, v := range missingMigrations {
		m, err := p.getMigration(v.versionID)
		if err != nil {
			return nil, err
		}
		apply = append(apply, m)
	}
	// filter all migrations with a version greater than the supplied version (min) and less than or
	// equal to the requested version (max). Skip any migrations that have already been applied.
	for _, m := range p.migrations {
		if dbAppliedVersions[m.Source.Version] {
			continue
		}
		if m.Source.Version > dbMaxVersion && m.Source.Version <= version {
			apply = append(apply, m)
		}
	}
	return apply, nil
}
