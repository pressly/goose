package goose

import (
	"context"
	"fmt"
)

// Down migrates the last known database version down.
func (p *Provider) Down(ctx context.Context) error {
	if p.opt.NoVersioning {
		currentVersion := p.migrations[len(p.migrations)-1].Version
		// Migrate only the latest migration down.
		return p.downToNoVersioning(ctx, currentVersion-1)
	}
	dbVersion, err := p.GetDBVersion(ctx)
	if err != nil {
		return err
	}
	migration, err := p.migrations.Current(dbVersion)
	if err != nil {
		return fmt.Errorf("failed to find migration:%d", dbVersion)
	}
	return p.startMigration(ctx, false, migration)
}

func (p *Provider) downToNoVersioning(ctx context.Context, version int64) error {
	for i := len(p.migrations) - 1; i >= 0; i-- {
		if version >= p.migrations[i].Version {
			return nil
		}
		if err := p.startMigration(ctx, false, p.migrations[i]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Provider) DownTo(ctx context.Context, version int64) error {
	if p.opt.NoVersioning {
		return p.downToNoVersioning(ctx, version)
	}

	// Down migrations always use the database version as a
	// reference which migrations to roll back
	// This becomes important when we have out of order migrations
	// Example 1,4 then 2,3,5 got applied. The db versions are:
	// 1,4,2,3,5

	// Should the down be 5,3,2,4,1
	// Or do we construct the version and walk it backwards
	// 5,4,3,2,1
	// I think the most accurate way is to migrate down based on the initial order that was applied.
	for {
		dbVersion, err := p.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		if dbVersion == 0 {
			return nil
		}
		current, err := p.migrations.Current(dbVersion)
		if err != nil {
			return err
		}
		if current.Version <= version {
			// TODO(mf): this logic ?
			return nil
		}
		if err := p.startMigration(ctx, false, current); err != nil {
			return err
		}
	}
}
