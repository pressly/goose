package goose

import "context"

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
	_ = currentVersion
	return nil
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
	return nil
}

// goose redo is the same as goose down followed by goose up-by-one. Reapplying the latest migration.
func (p *Provider) Redo(ctx context.Context) error {
	p.Down(ctx)
	p.UpByOne(ctx)
	return nil
}

// goose reset is the same as goose down-to 0. Applying all down migrations.
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
