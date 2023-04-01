package goose

import "context"

// Reset rolls back all migrations. It is an alias for DownTo(0).
func (p *Provider) Reset(ctx context.Context) ([]*MigrationResult, error) {
	return p.DownTo(ctx, 0)
}
