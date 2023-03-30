package goose

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pressly/goose/v4/internal/sqlparser"
)

func (p *Provider) up(ctx context.Context, upByOne bool, version int64) ([]*MigrationResult, error) {
	if version < 1 {
		return nil, fmt.Errorf("version must be a number greater than zero: %d", version)
	}

	conn, err := p.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// feat(mf): this is where a session level advisory lock would be acquired to ensure that only
	// one goose process is running at a time. Also need to lock the Provider itself with a mutex.
	// https://github.com/pressly/goose/issues/335

	if p.opt.NoVersioning {
		return p.runMigrations(ctx, conn, p.migrations, sqlparser.DirectionUp, upByOne)
	}

	if err := p.ensureVersionTable(ctx, conn); err != nil {
		return nil, err
	}

	dbMigrations, err := p.store.ListMigrationsConn(ctx, conn)
	if err != nil {
		return nil, err
	}
	currentVersion := dbMigrations[0].Version
	// lookupAppliedInDB is a map of all applied migrations in the database.
	lookupAppliedInDB := make(map[int64]bool)
	for _, m := range dbMigrations {
		lookupAppliedInDB[m.Version] = true
	}

	missingMigrations := findMissingMigrations(dbMigrations, p.migrations)

	// feature(mf): It is very possible someone may want to apply ONLY new migrations and skip
	// missing migrations entirely. At the moment this is not supported, but leaving this comment
	// because that's where that logic will be handled.
	if len(missingMigrations) > 0 && !p.opt.AllowMissing {
		var collected []string
		for _, v := range missingMigrations {
			collected = append(collected, strconv.FormatInt(v, 10))
		}
		msg := "migration"
		if len(collected) > 1 {
			msg += "s"
		}
		return nil, fmt.Errorf("found %d missing %s: %s",
			len(missingMigrations), msg, strings.Join(collected, ","))
	}

	var migrationsToApply []*migration
	if p.opt.AllowMissing {
		for _, v := range missingMigrations {
			m, err := p.getMigration(v)
			if err != nil {
				return nil, err
			}
			migrationsToApply = append(migrationsToApply, m)
		}
	}
	// filter all migrations with a version greater than the supplied version (min) and less than or
	// equal to the requested version (max).
	for _, m := range p.migrations {
		if lookupAppliedInDB[m.version] {
			continue
		}
		if m.version > currentVersion && m.version <= version {
			migrationsToApply = append(migrationsToApply, m)
		}
	}
	if len(migrationsToApply) == 0 {
		if upByOne {
			return nil, ErrNoNextVersion
		}
		return nil, nil
	}

	// feat(mf): this is where can (optionally) group multiple migrations to be run in a single
	// transaction. The default is to apply each migration sequentially on its own.
	// https://github.com/pressly/goose/issues/222
	//
	// Note, we can't use a single transaction for all migrations because some may have to be run in
	// their own transaction.

	return p.runMigrations(ctx, conn, migrationsToApply, sqlparser.DirectionUp, upByOne)
}
