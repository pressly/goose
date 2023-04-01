package goose

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v4/internal/sqlparser"
	"go.uber.org/multierr"
)

func (p *Provider) runMigrations(
	ctx context.Context,
	conn *sql.Conn,
	migrations []*migration,
	direction sqlparser.Direction,
	byOne bool,
) ([]*MigrationResult, error) {
	length := len(migrations)
	if byOne {
		length = 1
	}
	// Lazy parse SQL migrations (if any). We do this before running any migrations so that we can
	// fail fast if there are any errors and avoid leaving the database in a partially migrated
	// state.
	for _, m := range migrations {
		if m.migrationType == MigrationTypeSQL && !m.sqlParsed {
			parsedSQLMigration, err := parseSQL(p.opt.Filesystem, m.source, p.opt.Debug)
			if err != nil {
				return nil, err
			}
			m.sqlParsed = true
			m.sqlMigration = parsedSQLMigration
		}
	}
	results := make([]*MigrationResult, 0, length)
	for _, m := range migrations {
		start := time.Now()

		if err := p.runIndividually(ctx, conn, direction, m); err != nil {
			return nil, fmt.Errorf("failed to run %s migration: %s: %w",
				m.migrationType,
				filepath.Base(m.source),
				err,
			)
		}

		results = append(results, &MigrationResult{
			Migration: m.toMigration(),
			Duration:  time.Since(start),
		})
		if byOne && len(results) == 1 {
			break
		}
	}
	return results, nil
}

// runIndividually runs an individual migration, opening a new transaction if the migration is safe
// to run in a transaction. Otherwise, it runs the migration outside of a transaction with the
// supplied connection.
func (p *Provider) runIndividually(
	ctx context.Context,
	conn *sql.Conn,
	direction sqlparser.Direction,
	m *migration,
) error {
	switch m.migrationType {
	case MigrationTypeSQL:
		if m.sqlMigration.useTx {
			return p.runSQLBeginTx(ctx, conn, direction, m)
		} else {
			return p.runSQLNoTx(ctx, conn, direction, m)
		}
	case MigrationTypeGo:
		if m.goMigration.useTx {
			return p.runGoBeginTx(ctx, conn, direction, m)
		} else {
			// bug(mf): this is a potential deadlock scenario. We're running the Go migration with a
			// *sql.DB, but if/when we introduce locking (which will likely use *sql.Conn) AND if
			// the user set max open connections to 1, then this will deadlock.
			//
			// A potential solution is to expose a third Go register function *sql.Conn. Or continue
			// to use *sql.DB, but to use a separate connection pool for Go migrations and document
			// that the user should set max open connections greater than 1.
			//
			// In the Provider constructor we can also throw an error  when a user set max open
			// connections to 1 and has Go migrations that are registered to run outside of a
			// transaction.
			return p.runGoNoTx(ctx, direction, m)
		}
	}
	return fmt.Errorf("unknown migration type: %s", m.migrationType)
}

func (p *Provider) beginTx(
	ctx context.Context,
	conn *sql.Conn,
	direction sqlparser.Direction,
	version int64,
	fn func(tx *sql.Tx) error,
) (retErr error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			retErr = multierr.Append(retErr, tx.Rollback())
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	if !p.opt.NoVersioning {
		if err := p.store.InsertOrDelete(ctx, tx, direction.ToBool(), version); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (p *Provider) runGoBeginTx(
	ctx context.Context,
	conn *sql.Conn,
	direction sqlparser.Direction,
	m *migration,
) (retErr error) {
	return p.beginTx(ctx, conn, direction, m.version, func(tx *sql.Tx) error {
		fn := m.goMigration.downFn
		if direction == sqlparser.DirectionUp {
			fn = m.goMigration.upFn
		}
		if fn != nil {
			return fn(tx)
		}
		return nil
	})
}

func (p *Provider) runSQLBeginTx(
	ctx context.Context,
	conn *sql.Conn,
	direction sqlparser.Direction,
	m *migration,
) error {
	return p.beginTx(ctx, conn, direction, m.version, func(tx *sql.Tx) error {
		statements, err := m.getSQLStatements(direction)
		if err != nil {
			return err
		}
		for _, query := range statements {
			if _, err := tx.ExecContext(ctx, query); err != nil {
				return err
			}
		}
		return nil
	})
}

func (p *Provider) runSQLNoTx(
	ctx context.Context,
	conn *sql.Conn,
	direction sqlparser.Direction,
	m *migration,
) error {
	statements, err := m.getSQLStatements(direction)
	if err != nil {
		return err
	}
	for _, query := range statements {
		if _, err := conn.ExecContext(ctx, query); err != nil {
			return err
		}
	}
	if p.opt.NoVersioning {
		return nil
	}
	return p.store.InsertOrDeleteConn(ctx, conn, direction.ToBool(), m.version)
}

func (p *Provider) runGoNoTx(
	ctx context.Context,
	direction sqlparser.Direction,
	m *migration,
) error {
	fn := m.goMigration.downFnNoTx
	if direction == sqlparser.DirectionUp {
		fn = m.goMigration.upFnNoTx
	}
	if fn != nil {
		if err := fn(p.db); err != nil {
			return err
		}
	}
	if p.opt.NoVersioning {
		return nil
	}
	return p.store.InsertOrDeleteNoTx(ctx, p.db, direction.ToBool(), m.version)
}
