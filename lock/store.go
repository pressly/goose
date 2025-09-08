package lock

import (
	"context"
	"database/sql"
	"time"
)

// LockStore defines the interface for high-level lock operations.
// This abstracts the database-specific lock implementation details.
type LockStore interface {
	// TableName returns the name of the lock table.
	TableName() string
	
	// CreateTable creates the lock table if it doesn't exist.
	CreateTable(ctx context.Context, conn *sql.Conn) error
	
	// AcquireLock attempts to atomically acquire a lock.
	// Returns true if the lock was acquired, false if it's already held.
	AcquireLock(ctx context.Context, conn *sql.Conn, processInfo string) (bool, error)
	
	// ReleaseLock releases a held lock.
	// Returns true if the lock was successfully released.
	ReleaseLock(ctx context.Context, conn *sql.Conn) (bool, error)
	
	// UpdateHeartbeat updates the heartbeat timestamp for the current lock holder.
	UpdateHeartbeat(ctx context.Context, conn *sql.Conn) error
	
	// CleanupStaleLocks removes locks that haven't been updated within the stale timeout.
	CleanupStaleLocks(ctx context.Context, conn *sql.Conn, staleTimeout time.Duration) error
}

// LockResult represents the result of a lock operation.
type LockResult struct {
	Acquired  bool
	ProcessID string
	GrantedAt time.Time
}

// NewLockStore creates a new LockStore implementation using the provided LockQuerier.
func NewLockStore(tableName string, querier LockQuerier) LockStore {
	return &lockStore{
		tableName: tableName,
		querier:   querier,
	}
}

type lockStore struct {
	tableName string
	querier   LockQuerier
}

var _ LockStore = (*lockStore)(nil)

func (s *lockStore) TableName() string {
	return s.tableName
}

func (s *lockStore) CreateTable(ctx context.Context, conn *sql.Conn) error {
	query := s.querier.CreateLockTable(s.tableName)
	_, err := conn.ExecContext(ctx, query)
	return err
}

func (s *lockStore) AcquireLock(ctx context.Context, conn *sql.Conn, processInfo string) (bool, error) {
	// Wrap the entire lock acquisition in a transaction to ensure atomicity
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	
	// First try to acquire lock with UPDATE
	query := s.querier.AcquireLock(s.tableName)
	result, err := tx.ExecContext(ctx, query, processInfo)
	if err != nil {
		return false, err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	
	if rowsAffected == 1 {
		// Lock acquired via UPDATE, commit transaction
		return true, tx.Commit()
	}
	
	// No rows affected by UPDATE, try inserting initial row
	insertQuery := s.querier.InsertInitialLock(s.tableName)
	result, err = tx.ExecContext(ctx, insertQuery, processInfo)
	if err != nil {
		return false, err
	}
	
	rowsAffected, err = result.RowsAffected()
	if err != nil {
		return false, err
	}
	
	if rowsAffected == 1 {
		// Lock acquired via INSERT, commit transaction
		return true, tx.Commit()
	}
	
	// Neither UPDATE nor INSERT worked, someone else got the lock
	return false, nil
}

func (s *lockStore) ReleaseLock(ctx context.Context, conn *sql.Conn) (bool, error) {
	query := s.querier.ReleaseLock(s.tableName)
	result, err := conn.ExecContext(ctx, query)
	if err != nil {
		return false, err
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	
	return rowsAffected == 1, nil
}

func (s *lockStore) UpdateHeartbeat(ctx context.Context, conn *sql.Conn) error {
	query := s.querier.UpdateHeartbeat(s.tableName)
	_, err := conn.ExecContext(ctx, query)
	return err
}

func (s *lockStore) CleanupStaleLocks(ctx context.Context, conn *sql.Conn, staleTimeout time.Duration) error {
	query := s.querier.CleanupStaleLocks(s.tableName)
	staleTimeoutSeconds := int(staleTimeout.Seconds())
	_, err := conn.ExecContext(ctx, query, staleTimeoutSeconds)
	return err
}