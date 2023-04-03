package goose

import "context"

type RedoResult struct {
	DownResult *MigrationResult
	UpResult   *MigrationResult
}

// Redo rolls back the most recently applied migration, then runs it again.
//
// Important, it is not safe to run this function concurrently with other goose functions.
func (p *Provider) Redo(ctx context.Context) (*RedoResult, error) {
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
	return &RedoResult{
		DownResult: downResult,
		UpResult:   upResult,
	}, nil
}
