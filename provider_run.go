package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/sqlparser"
	"go.uber.org/multierr"
)

var (
	errMissingZeroVersion = errors.New("missing zero version migration")
)

func (p *Provider) resolveUpMigrations(
	dbVersions []*database.ListMigrationsResult,
	version int64,
) ([]*Migration, error) {
	var apply []*Migration
	var dbMaxVersion int64
	// dbAppliedVersions is a map of all applied migrations in the database.
	dbAppliedVersions := make(map[int64]bool, len(dbVersions))
	for _, m := range dbVersions {
		dbAppliedVersions[m.Version] = true
		if m.Version > dbMaxVersion {
			dbMaxVersion = m.Version
		}
	}
	missingMigrations := checkMissingMigrations(dbVersions, p.migrations)
	// feat(mf): It is very possible someone may want to apply ONLY new migrations and skip missing
	// migrations entirely. At the moment this is not supported, but leaving this comment because
	// that's where that logic would be handled.
	//
	// For example, if db has 1,4 applied and 2,3,5 are new, we would apply only 5 and skip 2,3. Not
	// sure if this is a common use case, but it's possible.
	if len(missingMigrations) > 0 && !p.cfg.allowMissing {
		var collected []string
		for _, v := range missingMigrations {
			collected = append(collected, fmt.Sprintf("%d", v.versionID))
		}
		msg := "migration"
		if len(collected) > 1 {
			msg += "s"
		}
		return nil, fmt.Errorf("found %d missing (out-of-order) %s lower than current max (%d): [%s]",
			len(missingMigrations), msg, dbMaxVersion, strings.Join(collected, ","),
		)
	}
	for _, v := range missingMigrations {
		m, err := p.getMigration(v.versionID)
		if err != nil {
			return nil, err
		}
		apply = append(apply, m)
	}
	// filter all migrations with a version greater than the supplied version (min) and less than or
	// equal to the requested version (max). Skip any migrations that have already been applied.
	for _, m := range p.migrations {
		if dbAppliedVersions[m.Version] {
			continue
		}
		if m.Version > dbMaxVersion && m.Version <= version {
			apply = append(apply, m)
		}
	}
	return apply, nil
}

func (p *Provider) prepareMigration(fsys fs.FS, m *Migration, direction bool) error {
	switch m.Type {
	case TypeGo:
		if m.goUp.Mode == 0 {
			return errors.New("go up migration mode is not set")
		}
		if m.goDown.Mode == 0 {
			return errors.New("go down migration mode is not set")
		}
		var useTx bool
		if direction {
			useTx = m.goUp.Mode == TransactionEnabled
		} else {
			useTx = m.goDown.Mode == TransactionEnabled
		}
		// bug(mf): this is a potential deadlock scenario. We're running Go migrations with *sql.DB,
		// but are locking the database with *sql.Conn. If the caller sets max open connections to
		// 1, then this will deadlock because the Go migration will try to acquire a connection from
		// the pool, but the pool is exhausted because the lock is held.
		//
		// A potential solution is to expose a third Go register function *sql.Conn. Or continue to
		// use *sql.DB and document that the user SHOULD NOT SET max open connections to 1. This is
		// a bit of an edge case. For now, we guard against this scenario by checking the max open
		// connections and returning an error.
		if p.cfg.lockEnabled && p.cfg.sessionLocker != nil && p.db.Stats().MaxOpenConnections == 1 {
			if !useTx {
				return errors.New("potential deadlock detected: cannot run Go migration without a transaction when max open connections set to 1")
			}
		}
		return nil
	case TypeSQL:
		if m.sql.Parsed {
			return nil
		}
		parsed, err := sqlparser.ParseAllFromFS(fsys, m.Source, false)
		if err != nil {
			return err
		}
		m.sql.Parsed = true
		m.sql.UseTx = parsed.UseTx
		m.sql.Up, m.sql.Down = parsed.Up, parsed.Down
		return nil
	}
	return fmt.Errorf("invalid migration type: %+v", m)
}

// runMigrations runs migrations sequentially in the given direction. If the migrations list is
// empty, return nil without error.
func (p *Provider) runMigrations(
	ctx context.Context,
	conn *sql.Conn,
	migrations []*Migration,
	direction sqlparser.Direction,
	byOne bool,
) ([]*MigrationResult, error) {
	if len(migrations) == 0 {
		return nil, nil
	}
	apply := migrations
	if byOne {
		apply = migrations[:1]
	}

	// SQL migrations are lazily parsed in both directions. This is done before attempting to run
	// any migrations to catch errors early and prevent leaving the database in an incomplete state.

	for _, m := range apply {
		if err := p.prepareMigration(p.fsys, m, direction.ToBool()); err != nil {
			return nil, err
		}
	}

	// feat(mf): If we decide to add support for advisory locks at the transaction level, this may
	// be a good place to acquire the lock. However, we need to be sure that ALL migrations are safe
	// to run in a transaction.

	// feat(mf): this is where we can (optionally) group multiple migrations to be run in a single
	// transaction. The default is to apply each migration sequentially on its own. See the
	// following issues for more details:
	//  - https://github.com/pressly/goose/issues/485
	//  - https://github.com/pressly/goose/issues/222

	var results []*MigrationResult
	for _, m := range apply {
		current := &MigrationResult{
			Source: Source{
				Type:    m.Type,
				Path:    m.Source,
				Version: m.Version,
			},
			Direction: direction.String(),
			Empty:     isEmpty(m, direction.ToBool()),
		}
		start := time.Now()
		if err := p.runIndividually(ctx, conn, m, direction.ToBool()); err != nil {
			// TODO(mf): we should also return the pending migrations here, the remaining items in
			// the apply slice.
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

func (p *Provider) runIndividually(
	ctx context.Context,
	conn *sql.Conn,
	m *Migration,
	direction bool,
) error {
	useTx, err := useTx(m, direction)
	if err != nil {
		return err
	}
	if useTx {
		return beginTx(ctx, conn, func(tx *sql.Tx) error {
			if err := runMigration(ctx, tx, m, direction); err != nil {
				return err
			}
			return p.maybeInsertOrDelete(ctx, tx, m.Version, direction)
		})
	}
	switch m.Type {
	case TypeGo:
		// Note, we are using *sql.DB instead of *sql.Conn because it's the Go migration contract.
		// This may be a deadlock scenario if max open connections is set to 1 AND a lock is
		// acquired on the database. In this case, the migration will block forever unable to
		// acquire a connection from the pool.
		//
		// For now, we guard against this scenario by checking the max open connections and
		// returning an error in the prepareMigration function.
		if err := runMigration(ctx, p.db, m, direction); err != nil {
			return err
		}
		return p.maybeInsertOrDelete(ctx, p.db, m.Version, direction)
	case TypeSQL:
		if err := runMigration(ctx, conn, m, direction); err != nil {
			return err
		}
		return p.maybeInsertOrDelete(ctx, conn, m.Version, direction)
	}
	return fmt.Errorf("failed to run individual migration: neither sql or go: %v", m)
}

func (p *Provider) maybeInsertOrDelete(
	ctx context.Context,
	db database.DBTxConn,
	version int64,
	direction bool,
) error {
	// If versioning is disabled, we don't need to insert or delete the migration version.
	if p.cfg.disableVersioning {
		return nil
	}
	if direction {
		return p.store.Insert(ctx, db, database.InsertRequest{Version: version})
	}
	return p.store.Delete(ctx, db, version)
}

// beginTx begins a transaction and runs the given function. If the function returns an error, the
// transaction is rolled back. Otherwise, the transaction is committed.
func beginTx(ctx context.Context, conn *sql.Conn, fn func(tx *sql.Tx) error) (retErr error) {
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
			//
			// TODO(mf): use [context.WithoutCancel] added in go1.21
			detachedCtx := context.Background()
			return multierr.Append(l.SessionUnlock(detachedCtx, conn), conn.Close())
		}
	}
	// If versioning is enabled, ensure the version table exists. For ad-hoc migrations, we don't
	// need the version table because there is no versioning.
	if !p.cfg.disableVersioning {
		if err := p.ensureVersionTable(ctx, conn); err != nil {
			return nil, nil, multierr.Append(err, cleanup())
		}
	}
	return conn, cleanup, nil
}

func (p *Provider) ensureVersionTable(ctx context.Context, conn *sql.Conn) (retErr error) {
	// feat(mf): this is where we can check if the version table exists instead of trying to fetch
	// from a table that may not exist. https://github.com/pressly/goose/issues/461
	res, err := p.store.GetMigration(ctx, conn, 0)
	if err == nil && res != nil {
		return nil
	}
	return beginTx(ctx, conn, func(tx *sql.Tx) error {
		if err := p.store.CreateVersionTable(ctx, tx); err != nil {
			return err
		}
		if p.cfg.disableVersioning {
			return nil
		}
		return p.store.Insert(ctx, tx, database.InsertRequest{Version: 0})
	})
}

type missingMigration struct {
	versionID int64
}

// checkMissingMigrations returns a list of migrations that are missing from the database. A missing
// migration is one that has a version less than the max version in the database.
func checkMissingMigrations(
	dbMigrations []*database.ListMigrationsResult,
	fsMigrations []*Migration,
) []missingMigration {
	existing := make(map[int64]bool)
	var dbMaxVersion int64
	for _, m := range dbMigrations {
		existing[m.Version] = true
		if m.Version > dbMaxVersion {
			dbMaxVersion = m.Version
		}
	}
	var missing []missingMigration
	for _, m := range fsMigrations {
		version := m.Version
		if !existing[version] && version < dbMaxVersion {
			missing = append(missing, missingMigration{
				versionID: version,
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
func (p *Provider) getMigration(version int64) (*Migration, error) {
	for _, m := range p.migrations {
		if m.Version == version {
			return m, nil
		}
	}
	return nil, ErrVersionNotFound
}
