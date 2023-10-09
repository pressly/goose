package lock

import (
	"time"
)

const (
	// DefaultLockID is the id used to lock the database for migrations. It is a crc64 hash of the
	// string "goose". This is used to ensure that the lock is unique to goose.
	//
	// crc64.Checksum([]byte("goose"), crc64.MakeTable(crc64.ECMA))
	DefaultLockID int64 = 5887940537704921958

	// Default values for the lock (time to wait for the lock to be acquired) and unlock (time to
	// wait for the lock to be released) durations.
	DefaultLockDuration   time.Duration = 60 * time.Minute
	DefaultUnlockDuration time.Duration = 1 * time.Minute
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

// WithLockDuration sets the max duration to wait for the lock to be acquired.
func WithLockDuration(duration time.Duration) SessionLockerOption {
	return sessionLockerConfigFunc(func(c *sessionLockerConfig) error {
		c.lockDuration = duration
		return nil
	})
}

// WithUnlockDuration sets the max duration to wait for the lock to be released.
func WithUnlockDuration(duration time.Duration) SessionLockerOption {
	return sessionLockerConfigFunc(func(c *sessionLockerConfig) error {
		c.unlockDuration = duration
		return nil
	})
}

type sessionLockerConfig struct {
	lockID         int64
	lockDuration   time.Duration
	unlockDuration time.Duration
}

var _ SessionLockerOption = (sessionLockerConfigFunc)(nil)

type sessionLockerConfigFunc func(*sessionLockerConfig) error

func (f sessionLockerConfigFunc) apply(cfg *sessionLockerConfig) error {
	return f(cfg)
}
