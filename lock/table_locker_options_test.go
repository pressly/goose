package lock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTableLockerOptions(t *testing.T) {
	// Test that options are applied correctly
	locker, err := NewPostgresTableLocker(
		WithTableName("custom_locks"),
		WithTableLockID(999),
		WithTableLeaseDuration(10*time.Second),
		WithTableHeartbeatInterval(3*time.Second),
	)
	require.NoError(t, err)
	require.NotNil(t, locker)
	// Test invalid lease duration
	_, err = NewPostgresTableLocker(WithTableLeaseDuration(-1 * time.Second))
	require.Error(t, err)
	// Test invalid heartbeat interval
	_, err = NewPostgresTableLocker(WithTableHeartbeatInterval(0))
	require.Error(t, err)
	// Test empty table name
	_, err = NewPostgresTableLocker(WithTableName(""))
	require.Error(t, err)
	// Test invalid lock ID
	_, err = NewPostgresTableLocker(WithTableLockID(0))
	require.Error(t, err)
	// Test invalid lock timeout interval duration
	_, err = NewPostgresTableLocker(WithTableLockTimeout(0, 10))
	require.Error(t, err)
	// Test invalid lock timeout failure threshold
	_, err = NewPostgresTableLocker(WithTableLockTimeout(5*time.Second, 0))
	require.Error(t, err)
	// Test invalid unlock timeout interval duration
	_, err = NewPostgresTableLocker(WithTableUnlockTimeout(0, 10))
	require.Error(t, err)
	// Test invalid unlock timeout failure threshold
	_, err = NewPostgresTableLocker(WithTableUnlockTimeout(5*time.Second, 0))
	require.Error(t, err)
}
