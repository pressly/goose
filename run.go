package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/pressly/goose/v4/internal/migration"
	"github.com/pressly/goose/v4/internal/sqlparser"
)

// MigrationResult is the result of a migration operation.
//
// Note, the caller is responsible for checking the Error field for any errors that occurred while
// running the migration. If the Error field is not nil, the migration failed.
type MigrationResult struct {
	// Full path to the migration file.
	Fullpath string
	// Version is the parsed version from the migration file name.
	Version int64
	// Duration is the time it took to run the migration.
	Duration time.Duration
	// Direction is the direction the migration was applied (up or down).
	Direction string
	// Empty is true if the file was valid, but no statements to apply in the given direction. These
	// are still tracked as applied migrations, but typically have no effect on the database.
	//
	// For SQL migrations, this means the file contained no statements. For Go migrations, this
	// means the file contained nil up or down functions.
	Empty bool

	// Error is any error that occurred while running the migration.
	Error error
}

// PartialError is returned when a migration fails, but some migrations already got applied.
type PartialError struct {
	// Results contains the results of all migrations that were applied before the error occurred.
	Results []*MigrationResult
	// Failed contains the result of the migration that failed.
	Failed *MigrationResult
	// Err is the error that occurred while running the migration.
	Err error
}

func (e *PartialError) Error() string {
	var filename string
	if e.Failed != nil {
		filename = fmt.Sprintf("(%s)", filepath.Base(e.Failed.Fullpath))
	} else {
		filename = "(file unknown)"
	}
	return fmt.Sprintf("partial migration error %s: %v", filename, e.Err)
}

// runMigrations runs migrations sequentially in the given direction.
//
// If the migrations slice is empty, this function returns nil with no error.
func (p *Provider) runMigrations(
	ctx context.Context,
	conn *sql.Conn,
	migrations []*migration.Migration,
	direction sqlparser.Direction,
	byOne bool,
) ([]*MigrationResult, error) {
	if len(migrations) == 0 {
		return nil, nil
	}
	var apply []*migration.Migration
	if byOne {
		apply = []*migration.Migration{migrations[0]}
	} else {
		apply = migrations
	}
	// Lazily parse SQL migrations (if any) in both directions. We do this before running any
	// migrations so that we can fail fast if there are any errors and avoid leaving the database in
	// a partially migrated state.
	if err := migration.ParseSQL(p.opt.Filesystem, p.opt.Debug, apply); err != nil {
		return nil, err
	}

	// TODO(mf): If we decide to add support for advisory locks at the transaction level, this may
	// be a good place to acquire the lock. However, we need to be sure that ALL migrations are safe
	// to run in a transaction.

	//
	//
	//

	// bug(mf): this is a potential deadlock scenario. We're running Go migrations with *sql.DB, but
	// are locking the database with *sql.Conn. If the caller sets max open connections to 1, then
	// this will deadlock because the Go migration will try to acquire a connection from the pool,
	// but the pool is locked.
	//
	// A potential solution is to expose a third Go register function *sql.Conn. Or continue to use
	// *sql.DB and document that the user SHOULD NOT SET max open connections to 1. This is a bit of
	// an edge case.
	if p.opt.LockMode != LockModeNone && p.db.Stats().MaxOpenConnections == 1 {
		for _, m := range apply {
			if m.IsGo() && !m.Go.UseTx {
				return nil, errors.New("potential deadlock detected: cannot run GoMigrationNoTx with max open connections set to 1")
			}
		}
	}

	// Run migrations individually, opening a new transaction for each migration if the migration is
	// safe to run in a transaction.

	// Avoid allocating a slice because we may have a partial migration error. 1. Avoid giving the
	// impression that N migrations were applied when in fact some were not 2. Avoid the caller
	// having to check for nil results
	var results []*MigrationResult
	for _, m := range apply {
		current := &MigrationResult{
			Fullpath:  m.Fullpath,
			Version:   m.Version,
			Direction: strings.ToLower(direction.String()),
			Empty:     m.IsEmpty(direction.ToBool()),
		}

		start := time.Now()
		if err := p.runIndividually(ctx, conn, direction.ToBool(), m); err != nil {
			current.Error = err
			current.Duration = time.Since(start)
			return nil, &PartialError{
				Results: results,
				Failed:  current,
				Err:     err,
			}
		}

		current.Duration = time.Since(start)
		results = append(results, current)
	}
	return results, nil
}

// runIndividually runs an individual migration, opening a new transaction if the migration is safe
// to run in a transaction. Otherwise, it runs the migration outside of a transaction with the
// supplied connection.
func (p *Provider) runIndividually(
	ctx context.Context,
	conn *sql.Conn,
	direction bool,
	m *migration.Migration,
) error {
	if m.UseTx() {
		// Run the migration in a transaction.
		return p.beginTx(ctx, conn, func(tx *sql.Tx) error {
			if err := m.Run(ctx, tx, direction); err != nil {
				return err
			}
			if p.opt.NoVersioning {
				return nil
			}
			return p.store.InsertOrDelete(ctx, tx, direction, m.Version)
		})
	}
	// Run the migration outside of a transaction.
	switch {
	case m.IsGo():
		// Note, we're using *sql.DB instead of *sql.Conn because it's the contract of the
		// GoMigrationNoTx function. This may be a deadlock scenario if the caller sets max open
		// connections to 1. See the comment in runMigrations for more details.
		if err := m.RunNoTx(ctx, p.db, direction); err != nil {
			return err
		}
	case m.IsSQL():
		if err := m.RunConn(ctx, conn, direction); err != nil {
			return err
		}
	}
	if p.opt.NoVersioning {
		return nil
	}
	return p.store.InsertOrDeleteConn(ctx, conn, direction, m.Version)
}

// beginTx begins a transaction and runs the given function. If the function returns an error, the
// transaction is rolled back. Otherwise, the transaction is committed.
//
// If the provider is configured to use versioning, this function also inserts or deletes the
// migration version.
func (p *Provider) beginTx(
	ctx context.Context,
	conn *sql.Conn,
	fn func(tx *sql.Tx) error,
) (retErr error) {
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if retErr != nil {
			retErr = errors.Join(retErr, tx.Rollback())
		}
	}()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (p *Provider) initialize(ctx context.Context) (*sql.Conn, func() error, error) {
	p.mu.Lock()

	conn, err := p.db.Conn(ctx)
	if err != nil {
		p.mu.Unlock()
		return nil, nil, err
	}
	var (
		// cleanup is a function that cleans up the connection, and optionally, the session lock.
		cleanup func() error
	)
	switch p.opt.LockMode {
	case LockModeAdvisorySession:
		if err := p.store.LockSession(ctx, conn); err != nil {
			p.mu.Unlock()
			return nil, nil, err
		}
		cleanup = func() error {
			defer p.mu.Unlock()
			return errors.Join(p.store.UnlockSession(ctx, conn), conn.Close())
		}
	case LockModeNone:
		cleanup = func() error {
			defer p.mu.Unlock()
			return conn.Close()
		}
	default:
		p.mu.Unlock()
		return nil, nil, fmt.Errorf("invalid lock mode: %d", p.opt.LockMode)
	}
	// If versioning is enabled, ensure the version table exists.
	//
	// For ad-hoc migrations, we don't need the version table because there is no versioning.
	if !p.opt.NoVersioning {
		if err := p.ensureVersionTable(ctx, conn); err != nil {
			return nil, nil, errors.Join(err, cleanup())
		}
	}
	return conn, cleanup, nil
}
