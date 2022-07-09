package goose

import "context"

// Redo rolls back the most recently applied migration (down), then runs it again (up).
func (p *Provider) Redo(ctx context.Context) error {
	var migration *Migration
	var err error
	if p.opt.NoVersioning {
		migration, err = p.migrations.Last()
		if err != nil {
			return err
		}
	} else {
		dbVersion, err := p.GetDBVersion(ctx)
		if err != nil {
			return err
		}
		migration, err = p.migrations.Current(dbVersion)
		if err != nil {
			return err
		}
	}
	if err := p.startMigration(ctx, false, migration); err != nil {
		return err
	}
	return p.startMigration(ctx, true, migration)
}
