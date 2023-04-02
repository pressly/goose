package goose

import (
	"context"

	"go.uber.org/multierr"
)

// GetDBVersion retrieves the current database version.
func (p *Provider) GetDBVersion(ctx context.Context) (_ int64, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()
	// Ensure version table exists.
	if err := p.ensureVersionTable(ctx, conn); err != nil {
		return 0, err
	}

	res, err := p.store.ListMigrationsConn(ctx, conn)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, nil
	}
	return res[0].Version, nil
}
