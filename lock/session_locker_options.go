package lock

import (
	"errors"
	"time"
)

const (
	// DefaultLockID is the id used to lock the database for migrations. It is a crc64 hash of the
	// string "goose". This is used to ensure that the lock is unique to goose.
	//
	// crc64.Checksum([]byte("goose"), crc64.MakeTable(crc64.ECMA))
	DefaultLockID int64 = 5887940537704921958
)

// SessionLockerOption is used to configure a SessionLocker.
type SessionLockerOption interface {
	apply(*sessionLockerConfig) error
}

// WithLockID sets the lock ID to use when locking the database.
//
// If WithLockID is not called, the DefaultLockID is used.
func WithLockID(lockID int64) SessionLockerOption {
	return sessionLockerConfigFunc(func(c *sessionLockerConfig) error {
		c.lockID = lockID
		return nil
	})
}

// WithLockTimeout sets the max duration to wait for the lock to be acquired. The total duration
// will be the period times the failure threshold.
//
// By default, the lock timeout is 300s (5min), where the lock is retried every 5 seconds (period)
// up to 60 times (failure threshold).
//
// The minimum period is 1 second, and the minimum failure threshold is 1.
func WithLockTimeout(period, failureThreshold uint64) SessionLockerOption {
	return sessionLockerConfigFunc(func(c *sessionLockerConfig) error {
		if period < 1 {
			return errors.New("period must be greater than 0, minimum is 1")
		}
		if failureThreshold < 1 {
			return errors.New("failure threshold must be greater than 0, minimum is 1")
		}
		c.lockProbe = probe{
			intervalDuration: time.Duration(period) * time.Second,
			failureThreshold: failureThreshold,
		}
		return nil
	})
}

// WithUnlockTimeout sets the max duration to wait for the lock to be released. The total duration
// will be the period times the failure threshold.
//
// By default, the lock timeout is 60s, where the lock is retried every 2 seconds (period) up to 30
// times (failure threshold).
//
// The minimum period is 1 second, and the minimum failure threshold is 1.
func WithUnlockTimeout(period, failureThreshold uint64) SessionLockerOption {
	return sessionLockerConfigFunc(func(c *sessionLockerConfig) error {
		if period < 1 {
			return errors.New("period must be greater than 0, minimum is 1")
		}
		if failureThreshold < 1 {
			return errors.New("failure threshold must be greater than 0, minimum is 1")
		}
		c.unlockProbe = probe{
			intervalDuration: time.Duration(period) * time.Second,
			failureThreshold: failureThreshold,
		}
		return nil
	})
}

// WithHeartbeatInterval sets how often to update the heartbeat timestamp when using table-based locking.
// This helps detect stale locks from crashed processes.
//
// By default, the heartbeat interval is 30 seconds.
//
// This option only affects table-based session lockers and is ignored by other implementations.
//
// The minimum interval is 1 second.
func WithHeartbeatInterval(interval time.Duration) SessionLockerOption {
	return sessionLockerConfigFunc(func(c *sessionLockerConfig) error {
		if interval < time.Second {
			return errors.New("heartbeat interval must be at least 1 second")
		}
		c.heartbeatInterval = interval
		return nil
	})
}

// WithStaleTimeout sets how long to wait before considering a lock stale and eligible for cleanup.
// When using table-based locking, if a lock's last heartbeat is older than this timeout, other
// processes can forcibly clean up and acquire the lock.
//
// By default, the stale timeout is 5 minutes.
//
// This option only affects table-based session lockers and is ignored by other implementations.
//
// The minimum timeout is 1 minute.
func WithStaleTimeout(timeout time.Duration) SessionLockerOption {
	return sessionLockerConfigFunc(func(c *sessionLockerConfig) error {
		if timeout < time.Minute {
			return errors.New("stale timeout must be at least 1 minute")
		}
		c.staleTimeout = timeout
		return nil
	})
}

type sessionLockerConfig struct {
	lockID            int64
	lockProbe         probe
	unlockProbe       probe
	heartbeatInterval time.Duration
	staleTimeout      time.Duration
}

// probe is used to configure how often and how many times to retry a lock or unlock operation. The
// total timeout will be the period times the failure threshold.
type probe struct {
	// How often (in seconds) to perform the probe.
	intervalDuration time.Duration
	// Number of times to retry the probe.
	failureThreshold uint64
}

var _ SessionLockerOption = (sessionLockerConfigFunc)(nil)

type sessionLockerConfigFunc func(*sessionLockerConfig) error

func (f sessionLockerConfigFunc) apply(cfg *sessionLockerConfig) error {
	return f(cfg)
}
