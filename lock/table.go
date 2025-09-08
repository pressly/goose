package lock

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/pressly/goose/v3/database"
	"github.com/sethvargo/go-retry"
)

// NewTableSessionLocker returns a SessionLocker that uses a database table for locking.
//
// Deprecated: Use NewTableSessionLockerForDialect instead to ensure optimal SQL queries
// for your specific database. This function requires a LockStore to be provided via
// NewTableSessionLockerWithStore.
//
// See [SessionLockerOption] for configuration options.
func NewTableSessionLocker(opts ...SessionLockerOption) (SessionLocker, error) {
	return nil, fmt.Errorf("NewTableSessionLocker requires a dialect-specific implementation, use NewTableSessionLockerForDialect or NewTableSessionLockerWithStore instead")
}

// NewTableSessionLockerForDialect creates a SessionLocker optimized for the specified database dialect.
// This ensures the most efficient SQL queries are used for the target database.
func NewTableSessionLockerForDialect(dialect database.Dialect, opts ...SessionLockerOption) (SessionLocker, error) {
	store, err := NewLockStoreForDialect(dialect, DefaultLockTableName)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock store for dialect %s: %w", dialect, err)
	}
	return NewTableSessionLockerWithStore(store, opts...)
}

// NewTableSessionLockerWithStore creates a SessionLocker with a custom LockStore.
// If store is nil, a generic LockStore will be created using the DefaultLockTableName.
func NewTableSessionLockerWithStore(store LockStore, opts ...SessionLockerOption) (SessionLocker, error) {
	cfg := sessionLockerConfig{
		lockID: DefaultLockID,
		lockProbe: probe{
			intervalDuration: 5 * time.Second,
			failureThreshold: 60,
		},
		unlockProbe: probe{
			intervalDuration: 2 * time.Second,
			failureThreshold: 30,
		},
		heartbeatInterval: 30 * time.Second,
		staleTimeout:      5 * time.Minute,
	}
	for _, opt := range opts {
		if err := opt.apply(&cfg); err != nil {
			return nil, err
		}
	}

	// If no store provided, return an error since we no longer support generic locking
	if store == nil {
		return nil, fmt.Errorf("LockStore is required, use NewTableSessionLockerForDialect or provide a store")
	}

	// Get hostname and process ID for lock identification
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	processInfo := fmt.Sprintf("%s:%d", hostname, os.Getpid())

	return &tableSessionLocker{
		lockID:      cfg.lockID,
		processInfo: processInfo,
		store:       store,
		retryLock: retry.WithMaxRetries(
			cfg.lockProbe.failureThreshold,
			retry.NewConstant(cfg.lockProbe.intervalDuration),
		),
		retryUnlock: retry.WithMaxRetries(
			cfg.unlockProbe.failureThreshold,
			retry.NewConstant(cfg.unlockProbe.intervalDuration),
		),
		heartbeatInterval: cfg.heartbeatInterval,
		staleTimeout:      cfg.staleTimeout,
	}, nil
}

type tableSessionLocker struct {
	lockID      int64
	processInfo string
	store       LockStore
	retryLock   retry.Backoff
	retryUnlock retry.Backoff

	// Heartbeat configuration
	heartbeatInterval time.Duration
	staleTimeout      time.Duration

	// Heartbeat control
	mu              sync.Mutex
	heartbeatCancel context.CancelFunc
	heartbeatConn   *sql.Conn
}

var _ SessionLocker = (*tableSessionLocker)(nil)

func (l *tableSessionLocker) SessionLock(ctx context.Context, conn *sql.Conn) error {
	return retry.Do(ctx, l.retryLock, func(ctx context.Context) error {
		// Create table if it doesn't exist
		if err := l.store.CreateTable(ctx, conn); err != nil {
			return fmt.Errorf("failed to create lock table: %w", err)
		}

		// Clean up stale locks before attempting to acquire
		if err := l.store.CleanupStaleLocks(ctx, conn, l.staleTimeout); err != nil {
			return fmt.Errorf("failed to cleanup stale locks: %w", err)
		}

		// Attempt atomic lock acquisition
		acquired, err := l.store.AcquireLock(ctx, conn, l.processInfo)
		if err != nil {
			return fmt.Errorf("failed to execute lock acquisition: %w", err)
		}

		if acquired {
			// Lock acquired successfully, start heartbeat
			if err := l.startHeartbeat(ctx, conn); err != nil {
				// If heartbeat fails to start, release the lock
				l.forceUnlock(ctx, conn)
				return fmt.Errorf("failed to start heartbeat: %w", err)
			}
			return nil
		}

		// Lock is held by another process, retry
		return retry.RetryableError(errors.New("failed to acquire lock"))
	})
}


func (l *tableSessionLocker) SessionUnlock(ctx context.Context, conn *sql.Conn) error {
	// Stop heartbeat first
	l.stopHeartbeat()

	return retry.Do(ctx, l.retryUnlock, func(ctx context.Context) error {
		released, err := l.store.ReleaseLock(ctx, conn)
		if err != nil {
			return fmt.Errorf("failed to execute lock release: %w", err)
		}

		if released {
			return nil
		}

		// Unexpected: no rows affected during unlock
		return retry.RetryableError(errors.New("failed to release lock"))
	})
}

func (l *tableSessionLocker) startHeartbeat(ctx context.Context, conn *sql.Conn) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Store the connection for heartbeat updates
	l.heartbeatConn = conn

	// Create context for heartbeat cancellation
	heartbeatCtx, cancel := context.WithCancel(ctx)
	l.heartbeatCancel = cancel

	// Start heartbeat goroutine
	go l.runHeartbeat(heartbeatCtx)

	return nil
}

func (l *tableSessionLocker) stopHeartbeat() {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.heartbeatCancel != nil {
		l.heartbeatCancel()
		l.heartbeatCancel = nil
		l.heartbeatConn = nil
	}
}

func (l *tableSessionLocker) runHeartbeat(ctx context.Context) {
	ticker := time.NewTicker(l.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Update heartbeat timestamp using the stored connection
			if l.heartbeatConn != nil {
				if err := l.store.UpdateHeartbeat(ctx, l.heartbeatConn); err != nil {
					// Log error but continue - the heartbeat will be detected as stale
					// and cleaned up by other processes
					continue
				}
			}
		}
	}
}

func (l *tableSessionLocker) forceUnlock(ctx context.Context, conn *sql.Conn) {
	// Best effort unlock without retries
	l.store.ReleaseLock(ctx, conn)
}

