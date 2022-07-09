package goose

import (
	"context"
)

// Reset applies all down migrations. This is equivalent to running DownTo 0.
func (p *Provider) Reset(ctx context.Context) error {
	return p.DownTo(ctx, 0)
}
