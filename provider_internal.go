package goose

import (
	"context"
	"fmt"
)

// Up migrates the database to the most recent version available.
func (p *Provider) Up(ctx context.Context) error {
	return p.up(ctx, false, maxVersion)
}

// UpByOne migrates the database by applying the next 1 version.
func (p *Provider) UpByOne(ctx context.Context) error {
	return p.up(ctx, true, maxVersion)
}

// UpTo migrates the database up to and including the supplied version.
//
// Example, we have 3 new versions available 9, 10, 11. The current
// database version is 8 and the requested version is 10. In this scenario
// versions 9 and 10 will be applied.
func (p *Provider) UpTo(ctx context.Context, version int64) error {
	return p.up(ctx, false, version)
}

// Down migrates the last known database version down.
func (p *Provider) Down(ctx context.Context) error {
	if p.opt.NoVersioning {
		currentVersion := p.migrations[len(p.migrations)-1].Version
		// Migrate only the latest migration down.
		return p.downToNoVersioning(ctx, currentVersion-1)
	}
	currentVersion, err := p.CurrentVersion(ctx)
	if err != nil {
		return err
	}
	migration, err := p.migrations.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("failed to find migration:%d", currentVersion)
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
		currentVersion, err := p.CurrentVersion(ctx)
		if err != nil {
			return err
		}
		if currentVersion == 0 {
			return nil
		}
		current, err := p.migrations.Current(currentVersion)
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

// Redo reapplies the last migration by migrating down and then up.
func (p *Provider) Redo(ctx context.Context) error {
	var migration *Migration
	var err error
	if p.opt.NoVersioning {
		migration, err = p.migrations.Last()
		if err != nil {
			return err
		}
	} else {
		currentVersion, err := p.CurrentVersion(ctx)
		if err != nil {
			return err
		}
		migration, err = p.migrations.Current(currentVersion)
		if err != nil {
			return err
		}
	}
	if err := p.startMigration(ctx, false, migration); err != nil {
		return err
	}
	return p.startMigration(ctx, true, migration)
}

// Reset applies all down migrations. This is equivalent to running DownTo 0.
func (p *Provider) Reset(ctx context.Context) error {
	return p.DownTo(ctx, 0)
}

// Ahhh, this is more of a "cli" command than a library command. All it does is
// print, and chances are users would want to control this behaviour. Printing
// should be left to the user.
func (p *Provider) Status(ctx context.Context) error {
	return Status(p.db, p.dir)
}

// replaces EnsureDBVersion && GetDBVersion ??
func (p *Provider) CurrentVersion(ctx context.Context) (int64, error) {
	var migrationRow migrationRow
	err := p.db.QueryRowContext(ctx, p.dialect.GetLatestMigration()).Scan(
		&migrationRow.ID,
		&migrationRow.VersionID,
		&migrationRow.Timestamp,
	)
	if err != nil {
		return 0, err
	}
	return migrationRow.VersionID, nil
}
