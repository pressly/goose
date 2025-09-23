package store

import (
	"context"
	"math/rand/v2"
	"strconv"
	"testing"
	"time"

	"github.com/pressly/goose/v3/internal/testing/testdb"
	"github.com/stretchr/testify/require"
)

func TestPostgresLockStore(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)

	ctx := context.Background()

	t.Run("StructuredReturnValues", func(t *testing.T) {
		store, err := NewPostgres("test_lock_store")
		require.NoError(t, err)

		lockID := rand.Int64()
		instanceID := "test-instance-" + strconv.Itoa(rand.IntN(100))
		leaseDuration := 10 * time.Second

		// Create the lock table first
		err = store.CreateLockTable(ctx, db)
		require.NoError(t, err)

		result, err := store.AcquireLock(ctx, db, lockID, instanceID, leaseDuration)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Lease expiration should be approximately now + lease duration
		require.Equal(t, instanceID, result.LockedBy)
		require.WithinDuration(t, time.Now().Add(leaseDuration), result.LeaseExpiresAt, 2*time.Second)

		newLeaseDuration := 15 * time.Second
		updateResult, err := store.UpdateLease(ctx, db, lockID, instanceID, newLeaseDuration)
		require.NoError(t, err)
		require.NotNil(t, updateResult)
		// Verify the lease was extended
		require.WithinDuration(t, time.Now().Add(newLeaseDuration), updateResult.LeaseExpiresAt, 2*time.Second, "updated lease expiration should reflect new duration")
		require.True(t, updateResult.LeaseExpiresAt.After(result.LeaseExpiresAt), "updated lease should be later than original")

		// Test ReleaseLock returns structured data
		releaseResult, err := store.ReleaseLock(ctx, db, lockID, instanceID)
		require.NoError(t, err)
		require.NotNil(t, releaseResult)

		// Verify the correct lock was released
		require.Equal(t, lockID, releaseResult.LockID, "returned lock ID should match the released lock")
	})

	t.Run("ErrorCases", func(t *testing.T) {
		// Create a PostgreSQL lock store
		store, err := NewPostgres("test_lock_errors")
		require.NoError(t, err)

		lockID := rand.Int64()
		instanceID1 := "instance-1"
		instanceID2 := "instance-2"
		leaseDuration := 5 * time.Second

		// Create the lock table first
		err = store.CreateLockTable(ctx, db)
		require.NoError(t, err)

		// First instance acquires the lock
		result1, err := store.AcquireLock(ctx, db, lockID, instanceID1, leaseDuration)
		require.NoError(t, err)
		require.Equal(t, instanceID1, result1.LockedBy)

		// Second instance tries to acquire the same lock - should fail
		_, err = store.AcquireLock(ctx, db, lockID, instanceID2, leaseDuration)
		require.Error(t, err)
		require.Contains(t, err.Error(), "already held by another instance")

		// Second instance tries to release a lock it doesn't own - should fail
		_, err = store.ReleaseLock(ctx, db, lockID, instanceID2)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not held by this instance")

		// Second instance tries to update lease it doesn't own - should fail
		_, err = store.UpdateLease(ctx, db, lockID, instanceID2, leaseDuration)
		require.Error(t, err)
		require.Contains(t, err.Error(), "not held by this instance")

		// First instance successfully releases its lock
		releaseResult, err := store.ReleaseLock(ctx, db, lockID, instanceID1)
		require.NoError(t, err)
		require.Equal(t, lockID, releaseResult.LockID)

		// Now second instance should be able to acquire it
		result2, err := store.AcquireLock(ctx, db, lockID, instanceID2, leaseDuration)
		require.NoError(t, err)
		require.Equal(t, instanceID2, result2.LockedBy)

		// Clean up
		_, err = store.ReleaseLock(ctx, db, lockID, instanceID2)
		require.NoError(t, err)
	})

}
