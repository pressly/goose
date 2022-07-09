package goose

import (
	"context"
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
