package table

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/pressly/goose/v3/lock/internal/store"
	"github.com/stretchr/testify/require"
)

// fakeLockStore is a store.LockStore that counts acquire and release attempts. It returns the
// configured error from each attempt.
type fakeLockStore struct {
	store.LockStore

	acquireErr   error
	acquireCalls int
	releaseErr   error
	releaseCalls int
}

func (s *fakeLockStore) CreateLockTable(context.Context, *sql.DB) error {
	return nil
}

func (s *fakeLockStore) AcquireLock(
	context.Context, *sql.DB, int64, string, time.Duration,
) (*store.AcquireLockResult, error) {
	s.acquireCalls++
	if s.acquireErr != nil {
		return nil, s.acquireErr
	}
	return &store.AcquireLockResult{}, nil
}

func (s *fakeLockStore) ReleaseLock(
	context.Context, *sql.DB, int64, string,
) (*store.ReleaseLockResult, error) {
	s.releaseCalls++
	if s.releaseErr != nil {
		return nil, s.releaseErr
	}
	return &store.ReleaseLockResult{}, nil
}

func (s *fakeLockStore) CleanupStaleLocks(context.Context, *sql.DB) ([]int64, error) {
	return nil, nil
}

const (
	testFailureThreshold = 3
	// One initial attempt, plus one attempt for each retry.
	testAttemptsPerCall = testFailureThreshold + 1
)

func newTestLocker(lockStore store.LockStore) *Locker {
	probe := ProbeConfig{
		IntervalDuration: time.Millisecond,
		FailureThreshold: testFailureThreshold,
	}
	return New(lockStore, Config{
		LockID:            1,
		LeaseDuration:     time.Minute,
		HeartbeatInterval: time.Hour,
		LockTimeout:       probe,
		UnlockTimeout:     probe,
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

// Each call must get the full number of retries. Retries that an earlier call used must not
// decrease the retries of a later call.
func TestLockRetriesOnEveryCall(t *testing.T) {
	t.Parallel()
	lockStore := &fakeLockStore{acquireErr: errors.New("already held by another instance")}
	locker := newTestLocker(lockStore)

	for call := 1; call <= 3; call++ {
		err := locker.Lock(t.Context(), nil)
		require.Error(t, err)
		require.Equal(t, call*testAttemptsPerCall, lockStore.acquireCalls, "call %d", call)
	}
}

func TestUnlockRetriesOnEveryCall(t *testing.T) {
	t.Parallel()
	lockStore := &fakeLockStore{releaseErr: errors.New("connection refused")}
	locker := newTestLocker(lockStore)

	for call := 1; call <= 3; call++ {
		require.NoError(t, locker.Lock(t.Context(), nil))
		err := locker.Unlock(t.Context(), nil)
		require.Error(t, err)
		require.Equal(t, call*testAttemptsPerCall, lockStore.releaseCalls, "call %d", call)
	}
}
