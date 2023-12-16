package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"runtime/debug"
	"sort"
	"strconv"
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
			collected = append(collected, strconv.FormatInt(v, 10))
		}
		msg := "migration"
		if len(collected) > 1 {
			msg += "s"
		}
		return nil, fmt.Errorf("found %d missing (out-of-order) %s lower than current max (%d): [%s]",
			len(missingMigrations), msg, dbMaxVersion, strings.Join(collected, ","),
		)
	}
	for _, missingVersion := range missingMigrations {
		m, err := p.getMigration(missingVersion)
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

// printf is a helper function that prints the given message if verbose is enabled. It also prepends
// the "goose: " prefix to the message.
func (p *Provider) printf(msg string, args ...interface{}) {
	if p.cfg.verbose {
		if !strings.HasPrefix(msg, "goose:") {
			msg = "goose: " + msg
		}
		p.cfg.logger.Printf(msg, args...)
	}
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
		if !p.cfg.disableVersioning {
			// No need to print this message if versioning is disabled because there are no
			// migrations being tracked in the goose version table.
			maxVersion, err := p.getDBMaxVersion(ctx, conn)
			if err != nil {
				return nil, err
			}
			p.printf("no migrations to run, current version: %d", maxVersion)
		}
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
			return nil, fmt.Errorf("failed to prepare migration %s: %w", m.ref(), err)
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
	//
	// Be careful, we can't use a single transaction for all migrations because some may be marked
	// as not using a transaction.

	var results []*MigrationResult
	for _, m := range apply {
		result := &MigrationResult{
			Source: &Source{
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
			result.Error = err
			result.Duration = time.Since(start)
			return nil, &PartialError{
				Applied: results,
				Failed:  result,
				Err:     err,
			}
		}
		result.Duration = time.Since(start)
		results = append(results, result)
		p.printf("%s", result)
	}
	if !p.cfg.disableVersioning && !byOne {
		maxVersion, err := p.getDBMaxVersion(ctx, conn)
		if err != nil {
			return nil, err
		}
		p.printf("successfully migrated database, current version: %d", maxVersion)
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
		// A lock was acquired, so we need to unlock the session when we're done. This is done by
		// returning a cleanup function that unlocks the session and closes the connection.
		cleanup = func() error {
			p.mu.Unlock()
			// Use a detached context to unlock the session. This is because the context passed to
			// SessionLock may have been canceled, and we don't want to cancel the unlock.
			//
			// TODO(mf): use context.WithoutCancel added in go1.21
			detachedCtx := context.Background()
			return multierr.Append(l.SessionUnlock(detachedCtx, conn), conn.Close())
		}
	}
	// If versioning is enabled, ensure the version table exists. For ad-hoc migrations, we don't
	// need the version table because no versions are being recorded.
	if !p.cfg.disableVersioning {
		if err := p.ensureVersionTable(ctx, conn); err != nil {
			return nil, nil, multierr.Append(err, cleanup())
		}
	}
	return conn, cleanup, nil
}

func (p *Provider) ensureVersionTable(ctx context.Context, conn *sql.Conn) (retErr error) {
	// existor is an interface that extends the Store interface with a method to check if the
	// version table exists. This API is not stable and may change in the future.
	type existor interface {
		TableExists(context.Context, database.DBTxConn, string) (bool, error)
	}
	if e, ok := p.store.(existor); ok {
		exists, err := e.TableExists(ctx, conn, p.store.Tablename())
		if err != nil {
			return fmt.Errorf("failed to check if version table exists: %w", err)
		}
		if exists {
			return nil
		}
	} else {
		// feat(mf): this is where we can check if the version table exists instead of trying to fetch
		// from a table that may not exist. https://github.com/pressly/goose/issues/461
		res, err := p.store.GetMigration(ctx, conn, 0)
		if err == nil && res != nil {
			return nil
		}
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

// checkMissingMigrations returns a list of migrations that are missing from the database. A missing
// migration is one that has a version less than the max version in the database.
func checkMissingMigrations(
	dbMigrations []*database.ListMigrationsResult,
	fsMigrations []*Migration,
) []int64 {
	existing := make(map[int64]bool)
	var dbMaxVersion int64
	for _, m := range dbMigrations {
		existing[m.Version] = true
		if m.Version > dbMaxVersion {
			dbMaxVersion = m.Version
		}
	}
	var missing []int64
	for _, m := range fsMigrations {
		if !existing[m.Version] && m.Version < dbMaxVersion {
			missing = append(missing, m.Version)
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		return missing[i] < missing[j]
	})
	return missing
}

// getMigration returns the migration for the given version. If no migration is found, then
// ErrVersionNotFound is returned.
func (p *Provider) getMigration(version int64) (*Migration, error) {
	for _, m := range p.migrations {
		if m.Version == version {
			return m, nil
		}
	}
	return nil, ErrVersionNotFound
}

// useTx is a helper function that returns true if the migration should be run in a transaction. It
// must only be called after the migration has been parsed and initialized.
func useTx(m *Migration, direction bool) (bool, error) {
	switch m.Type {
	case TypeGo:
		if m.goUp.Mode == 0 || m.goDown.Mode == 0 {
			return false, fmt.Errorf("go migrations must have a mode set")
		}
		if direction {
			return m.goUp.Mode == TransactionEnabled, nil
		}
		return m.goDown.Mode == TransactionEnabled, nil
	case TypeSQL:
		if !m.sql.Parsed {
			return false, fmt.Errorf("sql migrations must be parsed")
		}
		return m.sql.UseTx, nil
	}
	return false, fmt.Errorf("use tx: invalid migration type: %q", m.Type)
}

// isEmpty is a helper function that returns true if the migration has no functions or no statements
// to execute. It must only be called after the migration has been parsed and initialized.
func isEmpty(m *Migration, direction bool) bool {
	switch m.Type {
	case TypeGo:
		if direction {
			return m.goUp.RunTx == nil && m.goUp.RunDB == nil
		}
		return m.goDown.RunTx == nil && m.goDown.RunDB == nil
	case TypeSQL:
		if direction {
			return len(m.sql.Up) == 0
		}
		return len(m.sql.Down) == 0
	}
	return true
}

// runMigration is a helper function that runs the migration in the given direction. It must only be
// called after the migration has been parsed and initialized.
func runMigration(ctx context.Context, db database.DBTxConn, m *Migration, direction bool) error {
	switch m.Type {
	case TypeGo:
		return runGo(ctx, db, m, direction)
	case TypeSQL:
		return runSQL(ctx, db, m, direction)
	}
	return fmt.Errorf("invalid migration type: %q", m.Type)
}

// runGo is a helper function that runs the given Go functions in the given direction. It must only
// be called after the migration has been initialized.
func runGo(ctx context.Context, db database.DBTxConn, m *Migration, direction bool) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("panic: %v\n%s", r, debug.Stack())
		}
	}()

	switch db := db.(type) {
	case *sql.Conn:
		return fmt.Errorf("go migrations are not supported with *sql.Conn")
	case *sql.DB:
		if direction && m.goUp.RunDB != nil {
			return m.goUp.RunDB(ctx, db)
		}
		if !direction && m.goDown.RunDB != nil {
			return m.goDown.RunDB(ctx, db)
		}
		return nil
	case *sql.Tx:
		if direction && m.goUp.RunTx != nil {
			return m.goUp.RunTx(ctx, db)
		}
		if !direction && m.goDown.RunTx != nil {
			return m.goDown.RunTx(ctx, db)
		}
		return nil
	}
	return fmt.Errorf("invalid database connection type: %T", db)
}

// runSQL is a helper function that runs the given SQL statements in the given direction. It must
// only be called after the migration has been parsed.
func runSQL(ctx context.Context, db database.DBTxConn, m *Migration, direction bool) error {
	if !m.sql.Parsed {
		return fmt.Errorf("sql migrations must be parsed")
	}
	var statements []string
	if direction {
		statements = m.sql.Up
	} else {
		statements = m.sql.Down
	}
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
