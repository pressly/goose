package lock

import (
	"fmt"
	"time"

	"github.com/pressly/goose/v3/lock/internal/dialects"
	"github.com/pressly/goose/v3/lock/internal/store"
	"github.com/pressly/goose/v3/lock/internal/table"
)

// NewPostgresTableLocker returns a Locker that uses PostgreSQL table-based locking. It manages a
// single lock row and keeps the lock alive automatically.
//
// Default behavior:
//
//   - Lease (30s): How long the lock is valid if heartbeat stops
//   - Heartbeat (5s): How often the lock gets refreshed to keep it alive
//   - If the process dies, others can take the lock after lease expires
//
// Defaults:
//
//	Table: "goose_lock"
//	Lock ID: 5887940537704921958 (crc64 of "goose")
//	Lock retry: 5s intervals, 5min timeout
//	Unlock retry: 2s intervals, 1min timeout
//
// Lock and Unlock both retry on failure. Lock stays alive automatically until released. All
// defaults can be overridden with options.
func NewPostgresTableLocker(options ...TableLockerOption) (Locker, error) {
	config := table.Config{
		TableName:         DefaultLockTableName,
		LockID:            DefaultLockID,
		LeaseDuration:     30 * time.Second,
		HeartbeatInterval: 5 * time.Second,
		LockTimeout: table.ProbeConfig{
			IntervalDuration: 5 * time.Second,
			FailureThreshold: 60, // 5 minutes total
		},
		UnlockTimeout: table.ProbeConfig{
			IntervalDuration: 2 * time.Second,
			FailureThreshold: 30, // 1 minute total
		},
	}
	for _, opt := range options {
		if err := opt.apply(&config); err != nil {
			return nil, err
		}
	}
	// Create PostgreSQL querier
	querier := dialects.NewPostgresLockQuerier()
	// Create the lock store
	lockStore, err := store.New(config.TableName, querier)
	if err != nil {
		return nil, fmt.Errorf("create lock store: %w", err)
	}
	return table.New(lockStore, config), nil
}
