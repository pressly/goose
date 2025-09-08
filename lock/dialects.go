package lock

import (
	"fmt"

	"github.com/pressly/goose/v3/database"
)

// DefaultLockTableName is the default name for the lock table.
const DefaultLockTableName = "goose_db_lock"

// NewLockStoreForDialect creates a LockStore for the specified database dialect.
func NewLockStoreForDialect(dialect database.Dialect, tableName string) (LockStore, error) {
	if tableName == "" {
		tableName = DefaultLockTableName
	}
	
	var querier LockQuerier
	
	switch dialect {
	case database.DialectPostgres:
		querier = NewPostgresLockQuerier()
	case database.DialectSQLite3:
		querier = NewSQLiteLockQuerier()
	default:
		return nil, fmt.Errorf("table-based locking not implemented for dialect %q", dialect)
	}
	
	return NewLockStore(tableName, querier), nil
}


// PostgresLockQuerier provides PostgreSQL-specific lock queries.
type PostgresLockQuerier struct{}

func NewPostgresLockQuerier() LockQuerier {
	return &PostgresLockQuerier{}
}

func (q *PostgresLockQuerier) CreateLockTable(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY,
		locked INTEGER NOT NULL DEFAULT 0,
		lock_granted TIMESTAMP,
		last_heartbeat TIMESTAMP,
		locked_by TEXT
	)`, tableName)
}

func (q *PostgresLockQuerier) AcquireLock(tableName string) string {
	return fmt.Sprintf(`UPDATE %s 
		SET locked = 1, lock_granted = NOW(), last_heartbeat = NOW(), locked_by = $1
		WHERE id = 1 AND locked = 0`, tableName)
}

func (q *PostgresLockQuerier) InsertInitialLock(tableName string) string {
	return fmt.Sprintf(`INSERT INTO %s 
		(id, locked, lock_granted, last_heartbeat, locked_by) 
		VALUES (1, 1, NOW(), NOW(), $1) 
		ON CONFLICT (id) DO NOTHING`, tableName)
}

func (q *PostgresLockQuerier) ReleaseLock(tableName string) string {
	return fmt.Sprintf(`UPDATE %s 
		SET locked = 0, lock_granted = NULL, last_heartbeat = NULL, locked_by = NULL 
		WHERE id = 1`, tableName)
}

func (q *PostgresLockQuerier) UpdateHeartbeat(tableName string) string {
	return fmt.Sprintf(`UPDATE %s 
		SET last_heartbeat = NOW() 
		WHERE id = 1 AND locked = 1`, tableName)
}

func (q *PostgresLockQuerier) CleanupStaleLocks(tableName string) string {
	return fmt.Sprintf(`UPDATE %s 
		SET locked = 0, lock_granted = NULL, last_heartbeat = NULL, locked_by = NULL 
		WHERE id = 1 AND locked = 1 AND last_heartbeat < NOW() - INTERVAL '$1 seconds'`, tableName)
}


// SQLiteLockQuerier provides SQLite-specific lock queries.
type SQLiteLockQuerier struct{}

func NewSQLiteLockQuerier() LockQuerier {
	return &SQLiteLockQuerier{}
}

func (q *SQLiteLockQuerier) CreateLockTable(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		id INTEGER PRIMARY KEY,
		locked INTEGER NOT NULL DEFAULT 0,
		lock_granted TEXT,
		last_heartbeat TEXT,
		locked_by TEXT
	)`, tableName)
}

func (q *SQLiteLockQuerier) AcquireLock(tableName string) string {
	return fmt.Sprintf(`UPDATE %s 
		SET locked = 1, lock_granted = datetime('now'), last_heartbeat = datetime('now'), locked_by = ?
		WHERE id = 1 AND locked = 0`, tableName)
}

func (q *SQLiteLockQuerier) InsertInitialLock(tableName string) string {
	return fmt.Sprintf(`INSERT OR IGNORE INTO %s 
		(id, locked, lock_granted, last_heartbeat, locked_by) 
		VALUES (1, 1, datetime('now'), datetime('now'), ?)`, tableName)
}

func (q *SQLiteLockQuerier) ReleaseLock(tableName string) string {
	return fmt.Sprintf(`UPDATE %s 
		SET locked = 0, lock_granted = NULL, last_heartbeat = NULL, locked_by = NULL 
		WHERE id = 1`, tableName)
}

func (q *SQLiteLockQuerier) UpdateHeartbeat(tableName string) string {
	return fmt.Sprintf(`UPDATE %s 
		SET last_heartbeat = datetime('now') 
		WHERE id = 1 AND locked = 1`, tableName)
}

func (q *SQLiteLockQuerier) CleanupStaleLocks(tableName string) string {
	return fmt.Sprintf(`UPDATE %s 
		SET locked = 0, lock_granted = NULL, last_heartbeat = NULL, locked_by = NULL 
		WHERE id = 1 AND locked = 1 AND last_heartbeat < datetime('now', '-' || ? || ' seconds')`, tableName)
}
