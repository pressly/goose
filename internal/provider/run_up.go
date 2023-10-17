package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

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
	if p.cfg.noVersioning {
		// Short circuit if versioning is disabled and apply all migrations.
		return p.runMigrations(ctx, conn, p.migrations, sqlparser.DirectionUp, upByOne)
	}

	// optimize(mf): Listing all migrations from the database isn't great. This is only required to
	// support the out-of-order (allow missing) feature. For users who don't use this feature, we
	// could just query the database for the current version and then apply migrations that are
	// greater than that version.
	dbMigrations, err := p.store.ListMigrations(ctx, conn)
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
	if len(missingMigrations) > 0 && !p.cfg.allowMissing {
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

	var migrationsToApply []*migration
	if p.cfg.allowMissing {
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
		if lookupAppliedInDB[m.Source.Version] {
			continue
		}
		if m.Source.Version > dbMaxVersion && m.Source.Version <= version {
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
