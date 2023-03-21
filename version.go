package goose

import (
	"context"
	"errors"
)

// GetDBVersion retrieves the current database version.
//
// It is safe for concurrent use.
func (p *Provider) GetDBVersion(ctx context.Context) (_ int64, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		retErr = errors.Join(retErr, cleanup())
	}()

	res, err := p.store.ListMigrationsConn(ctx, conn)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, nil
	}
	return res[0].Version, nil
}
