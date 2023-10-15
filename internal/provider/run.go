package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/pressly/goose/v3/internal/sqladapter"
	"github.com/pressly/goose/v3/internal/sqlparser"
	"go.uber.org/multierr"
)

// runMigrations runs migrations sequentially in the given direction.
//
// If the migrations slice is empty, this function returns nil with no error.
func (p *Provider) runMigrations(
	ctx context.Context,
	conn *sql.Conn,
	migrations []*migration,
	direction sqlparser.Direction,
	byOne bool,
) ([]*MigrationResult, error) {
	if len(migrations) == 0 {
		return nil, nil
	}
	var apply []*migration
	if byOne {
		apply = []*migration{migrations[0]}
	} else {
		apply = migrations
	}
	// Lazily parse SQL migrations (if any) in both directions. We do this before running any
	// migrations so that we can fail fast if there are any errors and avoid leaving the database in
	// a partially migrated state.

	if err := parseSQL(p.fsys, false, apply); err != nil {
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
	// an edge case. if p.opt.LockMode != LockModeNone && p.db.Stats().MaxOpenConnections == 1 {
	//  for _, m := range apply {
	//      if m.IsGo() && !m.Go.UseTx {
	//          return nil, errors.New("potential deadlock detected: cannot run GoMigrationNoTx with max open connections set to 1")
	//      }
	//  }
	// }

	// Run migrations individually, opening a new transaction for each migration if the migration is
	// safe to run in a transaction.

	// Avoid allocating a slice because we may have a partial migration error. 1. Avoid giving the
	// impression that N migrations were applied when in fact some were not 2. Avoid the caller
	// having to check for nil results
	var results []*MigrationResult
	for _, m := range apply {
		current := &MigrationResult{
			Source:    m.Source,
			Direction: strings.ToLower(direction.String()),
			// TODO(mf): empty set here
		}

		start := time.Now()
		if err := p.runIndividually(ctx, conn, direction.ToBool(), m); err != nil {
			// TODO(mf): we should also return the pending migrations here.
			current.Error = err
			current.Duration = time.Since(start)
			return nil, &PartialError{
				Applied: results,
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
	m *migration,
) error {
	if m.useTx(direction) {
		// Run the migration in a transaction.
		return p.beginTx(ctx, conn, func(tx *sql.Tx) error {
			if err := m.run(ctx, tx, direction); err != nil {
				return err
			}
			if p.cfg.noVersioning {
				return nil
			}
			return p.store.InsertOrDelete(ctx, tx, direction, m.Source.Version)
		})
	}
	// Run the migration outside of a transaction.
	switch m.Source.Type {
	case TypeGo:
		// Note, we're using *sql.DB instead of *sql.Conn because it's the contract of the
		// GoMigrationNoTx function. This may be a deadlock scenario if the caller sets max open
		// connections to 1. See the comment in runMigrations for more details.
		if err := m.Go.runNoTx(ctx, p.db, direction); err != nil {
			return err
		}
	case TypeSQL:
		if err := m.runConn(ctx, conn, direction); err != nil {
			return err
		}
	}
	if p.cfg.noVersioning {
		return nil
	}
	return p.store.InsertOrDelete(ctx, conn, direction, m.Source.Version)
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
			retErr = multierr.Append(retErr, tx.Rollback())
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
	// cleanup is a function that cleans up the connection, and optionally, the session lock.
	cleanup := func() error {
		p.mu.Unlock()
		return conn.Close()
	}
	if l := p.cfg.sessionLocker; l != nil && p.cfg.lockEnabled {
		if err := l.SessionLock(ctx, conn); err != nil {
			return nil, nil, multierr.Append(err, cleanup())
		}
		cleanup = func() error {
			p.mu.Unlock()
			// Use a detached context to unlock the session. This is because the context passed to
			// SessionLock may have been canceled, and we don't want to cancel the unlock.
			// TODO(mf): use [context.WithoutCancel] added in go1.21
			detachedCtx := context.Background()
			return multierr.Append(l.SessionUnlock(detachedCtx, conn), conn.Close())
		}
	}
	// If versioning is enabled, ensure the version table exists. For ad-hoc migrations, we don't
	// need the version table because there is no versioning.
	if !p.cfg.noVersioning {
		if err := p.ensureVersionTable(ctx, conn); err != nil {
			return nil, nil, multierr.Append(err, cleanup())
		}
	}
	return conn, cleanup, nil
}

// parseSQL parses all SQL migrations in BOTH directions. If a migration has already been parsed, it
// will not be parsed again.
//
// Important: This function will mutate SQL migrations and is not safe for concurrent use.
func parseSQL(fsys fs.FS, debug bool, migrations []*migration) error {
	for _, m := range migrations {
		// If the migration is a SQL migration, and it has not been parsed, parse it.
		if m.Source.Type == TypeSQL && m.SQL == nil {
			parsed, err := sqlparser.ParseAllFromFS(fsys, m.Source.Fullpath, debug)
			if err != nil {
				return err
			}
			m.SQL = &sqlMigration{
				UseTx:          parsed.UseTx,
				UpStatements:   parsed.Up,
				DownStatements: parsed.Down,
			}
		}
	}
	return nil
}

func (p *Provider) ensureVersionTable(ctx context.Context, conn *sql.Conn) (retErr error) {
	// feat(mf): this is where we can check if the version table exists instead of trying to fetch
	// from a table that may not exist. https://github.com/pressly/goose/issues/461
	res, err := p.store.GetMigration(ctx, conn, 0)
	if err == nil && res != nil {
		return nil
	}
	return p.beginTx(ctx, conn, func(tx *sql.Tx) error {
		if err := p.store.CreateVersionTable(ctx, tx); err != nil {
			return err
		}
		if p.cfg.noVersioning {
			return nil
		}
		return p.store.InsertOrDelete(ctx, tx, true, 0)
	})
}

type missingMigration struct {
	versionID int64
	filename  string
}

// findMissingMigrations returns a list of migrations that are missing from the database. A missing
// migration is one that has a version less than the max version in the database.
func findMissingMigrations(
	dbMigrations []*sqladapter.ListMigrationsResult,
	fsMigrations []*migration,
	dbMaxVersion int64,
) []missingMigration {
	existing := make(map[int64]bool)
	for _, m := range dbMigrations {
		existing[m.Version] = true
	}
	var missing []missingMigration
	for _, m := range fsMigrations {
		version := m.Source.Version
		if !existing[version] && version < dbMaxVersion {
			missing = append(missing, missingMigration{
				versionID: version,
				filename:  m.filename(),
			})
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		return missing[i].versionID < missing[j].versionID
	})
	return missing
}

// getMigration returns the migration with the given version. If no migration is found, then
// ErrVersionNotFound is returned.
func (p *Provider) getMigration(version int64) (*migration, error) {
	for _, m := range p.migrations {
		if m.Source.Version == version {
			return m, nil
		}
	}
	return nil, ErrVersionNotFound
}

func (p *Provider) apply(ctx context.Context, version int64, direction bool) (_ *MigrationResult, retErr error) {
	if version < 1 {
		return nil, errors.New("version must be greater than zero")
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

	result, err := p.store.GetMigration(ctx, conn, version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	// If the migration has already been applied, return an error, unless the migration is being
	// applied in the opposite direction. In that case, we allow the migration to be applied again.
	if result != nil && direction {
		return nil, fmt.Errorf("version %d: %w", version, ErrAlreadyApplied)
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
		return nil, fmt.Errorf("version %d: %w", version, ErrAlreadyApplied)
	}
	return results[0], nil
}

func (p *Provider) status(ctx context.Context) (_ []*MigrationStatus, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	// TODO(mf): add support for limit and order. Also would be nice to refactor the list query to
	// support limiting the set.

	status := make([]*MigrationStatus, 0, len(p.migrations))
	for _, m := range p.migrations {
		migrationStatus := &MigrationStatus{
			Source: m.Source,
			State:  StatePending,
		}
		dbResult, err := p.store.GetMigration(ctx, conn, m.Source.Version)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if dbResult != nil {
			migrationStatus.State = StateApplied
			migrationStatus.AppliedAt = dbResult.Timestamp
		}
		status = append(status, migrationStatus)
	}

	return status, nil
}

func (p *Provider) getDBVersion(ctx context.Context) (_ int64, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		retErr = multierr.Append(retErr, cleanup())
	}()

	res, err := p.store.ListMigrations(ctx, conn)
	if err != nil {
		return 0, err
	}
	if len(res) == 0 {
		return 0, nil
	}
	return res[0].Version, nil
}
