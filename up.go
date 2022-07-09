package goose

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

func (p *Provider) up(ctx context.Context, upByOne bool, version int64) error {
	if version < 1 {
		return fmt.Errorf("version must be a number greater than zero: %d", version)
	}
	if p.opt.NoVersioning {
		// This code path does not rely on database state to resolve which
		// migrations have already been applied. Instead we blindly apply
		// the requested migrations when user requests no versioning.
		if upByOne {
			// For non-versioned up-by-one this means applying the first
			// migration over and over.
			version = p.migrations[0].Version
		}
		return p.upToNoVersioning(ctx, version)
	}

	dbMigrations, err := p.listAllDBMigrations(ctx)
	if err != nil {
		return err
	}
	missingMigrations := findMissingMigrations(dbMigrations, p.migrations)

	// feature(mf): It is very possible someone may want to apply ONLY new migrations
	// and skip missing migrations altogether. At the moment this is not supported,
	// but leaving this comment because that's where that logic will be handled.
	if len(missingMigrations) > 0 && !p.opt.AllowMissing {
		var collected []string
		for _, m := range missingMigrations {
			output := fmt.Sprintf("version %d: %s", m.Version, m.Source)
			collected = append(collected, output)
		}
		return fmt.Errorf("error: found %d missing migrations:\n\t%s",
			len(missingMigrations), strings.Join(collected, "\n\t"))
	}
	if p.opt.AllowMissing {
		return p.upAllowMissing(ctx, upByOne, missingMigrations, dbMigrations)
	}

	var current int64
	for {
		var err error
		current, err = p.CurrentVersion(ctx)
		if err != nil {
			return err
		}
		next, err := p.migrations.Next(current)
		if err != nil {
			if errors.Is(err, ErrNoNextVersion) {
				break
			}
			return fmt.Errorf("failed to find next migration: %w", err)
		}
		// TODO(mf): confirm this behavior. We used to limit this
		// by collecting only a subset of migration files and so
		// when we got to this loop we were only operating on the
		// target version. Now, we're collecting ALL migration
		// files and so we need to account for there always being
		// entire set of them.
		if next.Version > version {
			return nil
		}
		if err := p.startMigration(ctx, true, next); err != nil {
			return err
		}
		if upByOne {
			return nil
		}
	}
	// At this point there are no more migrations to apply. But we need to maintain
	// the following behaviour:
	// UpByOne returns an error to signifying there are no more migrations.
	// Up and UpTo return nil
	if upByOne {
		return ErrNoNextVersion
	}
	return nil
}

func (p *Provider) upAllowMissing(
	ctx context.Context,
	upByOne bool,
	missingMigrations Migrations,
	dbMigrations Migrations,
) error {
	lookupApplied := make(map[int64]bool)
	for _, found := range dbMigrations {
		lookupApplied[found.Version] = true
	}
	// Apply all missing migrations first.
	for _, missing := range missingMigrations {
		if err := p.startMigration(ctx, true, missing); err != nil {
			return err
		}
		// Apply one migration and return early.
		if upByOne {
			return nil
		}
		// TODO(mf): do we need this check? It's a bit redundant, but we may
		// want to keep it as a safe-guard. Maybe we should instead have
		// the underlying query (if possible) return the current version as
		// part of the same transaction.
		currentVersion, err := p.CurrentVersion(ctx)
		if err != nil {
			return err
		}
		if currentVersion != missing.Version {
			return fmt.Errorf("error: missing migration:%d does not match current db version:%d",
				currentVersion, missing.Version)
		}

		lookupApplied[missing.Version] = true
	}
	// We can no longer rely on the database version_id to be sequential because
	// missing (out-of-order) migrations get applied before newer migrations.
	for _, found := range p.migrations {
		// TODO(mf): instead of relying on this lookup, consider hitting
		// the database directly?
		// Alternatively, we can skip a bunch migrations and start the cursor
		// at a version that represents 100% applied migrations. But this is
		// risky, and we should aim to keep this logic simple.
		if lookupApplied[found.Version] {
			continue
		}
		if err := p.startMigration(ctx, true, found); err != nil {
			return err
		}
		if upByOne {
			return nil
		}
	}
	// At this point there are no more migrations to apply. But we need to maintain
	// the following behaviour:
	// UpByOne returns an error to signifying there are no more migrations.
	// Up and UpTo return nil
	if upByOne {
		return ErrNoNextVersion
	}
	return nil
}

func (p *Provider) upToNoVersioning(ctx context.Context, version int64) error {
	for _, current := range p.migrations {
		if current.Version > version {
			return nil
		}
		if err := p.startMigration(ctx, true, current); err != nil {
			return err
		}
	}
	return nil
}
