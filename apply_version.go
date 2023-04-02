package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pressly/goose/v4/internal/sqlparser"
	"go.uber.org/multierr"
)

// ApplyVersion applies exactly one migration at the specified version. If a migration cannot be
// found for the specified version, this method returns ErrNoCurrentVersion. If the migration has
// been applied already, this method returns ErrAlreadyApplied.
//
// If the direction is true, the migration will be applied. If the direction is false, the migration
// will be rolled back.
func (p *Provider) ApplyVersion(ctx context.Context, version int64, direction bool) (_ *MigrationResult, retErr error) {
	if version < 1 {
		return nil, fmt.Errorf("invalid version: %d", version)
	}

	m, err := p.getMigration(version)
	if err != nil {
		return nil, err
	}

	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()
	// Ensure version table exists.
	if err := p.ensureVersionTable(ctx, conn); err != nil {
		return nil, err
	}

	result, err := p.store.GetMigration(ctx, conn, version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if result != nil {
		return nil, ErrAlreadyApplied
	}

	d := sqlparser.DirectionDown
	if direction {
		d = sqlparser.DirectionUp
	}
	results, err := p.runMigrations(ctx, conn, []*migration{m}, d, true)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, ErrAlreadyApplied
	}
	return results[0], nil
}
