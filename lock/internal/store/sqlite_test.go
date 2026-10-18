package store

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

func newSQLiteDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := filepath.Join(t.TempDir(), "lock.db") + "?_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func TestSQLiteStore(t *testing.T) {
	t.Parallel()

	t.Run("empty_table_name", func(t *testing.T) {
		t.Parallel()
		_, err := NewSQLite("")
		require.Error(t, err)
	})
	t.Run("create_lock_table", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newSQLiteDB(t)
		for _, name := range []string{"goose_lock", "main.custom_lock"} {
			s, err := NewSQLite(name)
			require.NoError(t, err)
			exists, err := s.TableExists(ctx, db)
			require.NoError(t, err)
			require.False(t, exists, name)
			require.NoError(t, s.CreateLockTable(ctx, db))
			// Creating the table again must be a no-op.
			require.NoError(t, s.CreateLockTable(ctx, db))
			exists, err = s.TableExists(ctx, db)
			require.NoError(t, err)
			require.True(t, exists, name)
		}
	})
	t.Run("lock_lifecycle", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newSQLiteDB(t)
		s, err := NewSQLite("goose_lock")
		require.NoError(t, err)
		require.NoError(t, s.CreateLockTable(ctx, db))
		const lockID = 42

		_, err = s.CheckLockStatus(ctx, db, lockID)
		require.Error(t, err)

		before := time.Now()
		acquired, err := s.AcquireLock(ctx, db, lockID, "instance-a", time.Minute)
		require.NoError(t, err)
		require.Equal(t, "instance-a", acquired.LockedBy)
		requireAround(t, before.Add(time.Minute), acquired.LeaseExpiresAt)

		status, err := s.CheckLockStatus(ctx, db, lockID)
		require.NoError(t, err)
		require.True(t, status.Locked)
		require.NotNil(t, status.LockedBy)
		require.Equal(t, "instance-a", *status.LockedBy)
		require.NotNil(t, status.LeaseExpiresAt)
		require.True(t, acquired.LeaseExpiresAt.Equal(*status.LeaseExpiresAt))
		require.NotNil(t, status.UpdatedAt)
		requireAround(t, before, *status.UpdatedAt)

		// The lease has not expired, so another instance must not take the lock.
		_, err = s.AcquireLock(ctx, db, lockID, "instance-b", time.Minute)
		require.ErrorContains(t, err, "already held by another instance")
		_, err = s.UpdateLease(ctx, db, lockID, "instance-b", time.Minute)
		require.ErrorContains(t, err, "not held by this instance")
		_, err = s.ReleaseLock(ctx, db, lockID, "instance-b")
		require.ErrorContains(t, err, "not held by this instance")
		cleaned, err := s.CleanupStaleLocks(ctx, db)
		require.NoError(t, err)
		require.Empty(t, cleaned)

		// A different lock ID is independent.
		_, err = s.AcquireLock(ctx, db, lockID+1, "instance-b", time.Minute)
		require.NoError(t, err)

		renewed, err := s.UpdateLease(ctx, db, lockID, "instance-a", time.Hour)
		require.NoError(t, err)
		requireAround(t, before.Add(time.Hour), renewed.LeaseExpiresAt)

		released, err := s.ReleaseLock(ctx, db, lockID, "instance-a")
		require.NoError(t, err)
		require.EqualValues(t, lockID, released.LockID)
		status, err = s.CheckLockStatus(ctx, db, lockID)
		require.NoError(t, err)
		require.False(t, status.Locked)
		require.Nil(t, status.LockedBy)
		require.Nil(t, status.LeaseExpiresAt)

		// Once released, another instance can take the lock.
		acquired, err = s.AcquireLock(ctx, db, lockID, "instance-b", time.Minute)
		require.NoError(t, err)
		require.Equal(t, "instance-b", acquired.LockedBy)
	})
	t.Run("expired_lease", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		db := newSQLiteDB(t)
		s, err := NewSQLite("goose_lock")
		require.NoError(t, err)
		require.NoError(t, s.CreateLockTable(ctx, db))

		_, err = s.AcquireLock(ctx, db, 1, "instance-a", 50*time.Millisecond)
		require.NoError(t, err)
		_, err = s.AcquireLock(ctx, db, 2, "instance-a", 50*time.Millisecond)
		require.NoError(t, err)
		_, err = s.AcquireLock(ctx, db, 3, "instance-a", time.Hour)
		require.NoError(t, err)
		time.Sleep(200 * time.Millisecond)

		// An expired lease can be taken over directly.
		acquired, err := s.AcquireLock(ctx, db, 1, "instance-b", time.Minute)
		require.NoError(t, err)
		require.Equal(t, "instance-b", acquired.LockedBy)
		// The previous holder no longer owns it.
		_, err = s.UpdateLease(ctx, db, 1, "instance-a", time.Minute)
		require.Error(t, err)

		// Only the expired lock 2 is stale; lock 1 was re-acquired and lock 3 is still leased.
		cleaned, err := s.CleanupStaleLocks(ctx, db)
		require.NoError(t, err)
		require.Equal(t, []int64{2}, cleaned)
		status, err := s.CheckLockStatus(ctx, db, 2)
		require.NoError(t, err)
		require.False(t, status.Locked)
		status, err = s.CheckLockStatus(ctx, db, 3)
		require.NoError(t, err)
		require.True(t, status.Locked)
	})
}

// requireAround asserts that got is within a few seconds of want. Timestamps come from the database
// clock, which may differ slightly from the test clock.
func requireAround(t *testing.T, want, got time.Time) {
	t.Helper()
	require.WithinDuration(t, want, got, 5*time.Second)
}
