package goose

import "context"

// Redo rolls back the most recently applied migration, then runs it again.
//
// Important, it is not safe to run this function concurrently with other goose functions.
func (p *Provider) Redo(ctx context.Context) ([]*MigrationResult, error) {
	// feat(mf): lock the database to prevent concurrent migrations. Each of these functions should
	// be run within the same lock?
	downResult, err := p.Down(ctx)
	if err != nil {
		return nil, err
	}
	upResult, err := p.UpByOne(ctx)
	if err != nil {
		return nil, err
	}
	return []*MigrationResult{
		downResult,
		upResult,
	}, nil
}
