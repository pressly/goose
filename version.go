package goose

import "context"

// GetDBVersion retrieves the current database version.
func (p *Provider) GetDBVersion(ctx context.Context) (int64, error) {
	conn, err := p.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

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
