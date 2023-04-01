package goose

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pressly/goose/v4/internal/dialectadapter"
	"github.com/pressly/goose/v4/internal/sqlparser"
	"go.uber.org/multierr"
)

// Up applies all available migrations. If there are no migrations to apply, this method returns
// empty list and nil error.
func (p *Provider) Up(ctx context.Context) ([]*MigrationResult, error) {
	return p.up(ctx, false, math.MaxInt64)
}

// UpByOne applies the next available migration. If there are no migrations to apply, this method
// returns ErrNoNextVersion.
func (p *Provider) UpByOne(ctx context.Context) (*MigrationResult, error) {
	res, err := p.up(ctx, true, math.MaxInt64)
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, ErrNoNextVersion
	}
	return res[0], nil
}

// UpTo applies all available migrations up to and including the specified version. If there are no
// migrations to apply, this method returns empty list and nil error.
//
// For example, suppose there are 3 new migrations available 9, 10, 11. The current database version
// is 8 and the requested version is 10. In this scenario only versions 9 and 10 will be applied.
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
		retErr = multierr.Append(retErr, cleanup())
	}()
	// Ensure version table exists.
	if err := p.ensureVersionTable(ctx, conn); err != nil {
		return nil, err
	}

	if p.opt.NoVersioning {
		return p.runMigrations(ctx, conn, p.migrations, sqlparser.DirectionUp, upByOne)
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

// findMissingMigrations returns a list of migrations that are missing from the database. A missing
// migration is one that has a version less than the max version in the database.
func findMissingMigrations(
	dbMigrations []*dialectadapter.ListMigrationsResult,
	fsMigrations []*migration,
) []int64 {
	existing := make(map[int64]bool)
	var max int64
	for _, m := range dbMigrations {
		existing[m.Version] = true
		if m.Version > max {
			max = m.Version
		}
	}
	var missing []int64
	for _, m := range fsMigrations {
		if !existing[m.version] && m.version < max {
			missing = append(missing, m.version)
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		return missing[i] < missing[j]
	})
	return missing
}
