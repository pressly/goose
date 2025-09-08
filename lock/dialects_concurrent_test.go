package lock

import (
	"context"
	"database/sql"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3/database"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

// TestDialectConcurrentLocking tests that exactly one process wins the lock across all supported dialects
func TestDialectConcurrentLocking(t *testing.T) {
	dialects := []struct {
		name    string
		dialect database.Dialect
		dsn     func() string
	}{
		{
			name:    "postgres",
			dialect: database.DialectPostgres,
			dsn:     func() string { return testPostgresDB(t) },
		},
		{
			name:    "sqlite",
			dialect: database.DialectSQLite3,
			dsn:     func() string { return testSQLiteDB(t) },
		},
	}

	for _, d := range dialects {
		t.Run(d.name, func(t *testing.T) {
			if d.name == "postgres" && !isPostgresAvailable() {
				t.Skip("PostgreSQL not available")
			}

			testDialectConcurrentLockingImpl(t, d.dialect, d.dsn())
		})
	}
}

func testDialectConcurrentLockingImpl(t *testing.T, dialect database.Dialect, dsn string) {
	numGoroutines := 5 // Same as existing table tests

	t.Run("exactly_one_winner", func(t *testing.T) {
		db, err := sql.Open(getDriverName(dialect), dsn)
		require.NoError(t, err)
		defer db.Close()
		
		// Configure SQLite for proper concurrency  
		if dialect == database.DialectSQLite3 {
			_, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA locking_mode=NORMAL; PRAGMA synchronous=NORMAL;")
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		var mu sync.Mutex
		var successes, failures int

		ctx := context.Background()

		// Launch concurrent lock attempts
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				// Create individual locker instance per worker like working table tests
				var locker SessionLocker
				var err error
				if dialect == database.DialectSQLite3 {
					store := NewLockStore(DefaultLockTableName, NewSQLiteLockQuerier())
					locker, err = NewTableSessionLockerWithStore(store, 
						WithLockTimeout(1, 2), // Quick timeout like working tests
						WithHeartbeatInterval(time.Second),
					)
				} else {
					locker, err = NewTableSessionLockerForDialect(dialect,
						WithLockTimeout(1, 2), // Quick timeout like working tests
						WithHeartbeatInterval(time.Second),
					)
				}
				if err != nil {
					t.Errorf("goroutine %d: failed to create locker: %v", id, err)
					return
				}

				conn, err := db.Conn(ctx)
				if err != nil {
					t.Errorf("goroutine %d: failed to get connection: %v", id, err)
					return
				}
				defer conn.Close()

				err = locker.SessionLock(ctx, conn)
				
				mu.Lock()
				if err != nil {
					failures++
				} else {
					successes++
					// Hold lock briefly then release
					time.Sleep(100 * time.Millisecond)
					locker.SessionUnlock(ctx, conn)
				}
				mu.Unlock()
			}(i)
		}

		wg.Wait()

		// Exactly one should succeed, others should fail
		require.Equal(t, 1, successes, "exactly one goroutine should win the lock")
		require.Equal(t, numGoroutines-1, failures, "all other goroutines should fail")
	})

	t.Run("sequential_lock_acquisition", func(t *testing.T) {
		db, err := sql.Open(getDriverName(dialect), dsn)
		require.NoError(t, err)
		defer db.Close()
		
		// Configure SQLite for proper concurrency
		if dialect == database.DialectSQLite3 {
			_, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA locking_mode=NORMAL; PRAGMA synchronous=NORMAL;")
			require.NoError(t, err)
		}

		locker, err := NewTableSessionLockerForDialect(
			dialect,
			WithLockTimeout(1, 40), // 1 second intervals, 40 attempts = 40 second timeout
		)
		require.NoError(t, err)

		var successfulLocks int64
		var mu sync.Mutex
		var wg sync.WaitGroup

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		// Launch workers that will acquire locks sequentially
		numWorkers := 5
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				conn, err := db.Conn(ctx)
				if err != nil {
					t.Errorf("worker %d: failed to get connection: %v", id, err)
					return
				}
				defer conn.Close()

				// Keep trying to get the lock
				for {
					select {
					case <-ctx.Done():
						return
					default:
					}

					if err := locker.SessionLock(ctx, conn); err == nil {
						// Got the lock, do some work
						mu.Lock()
						successfulLocks++
						mu.Unlock()

						// Simulate work
						time.Sleep(200 * time.Millisecond)

						// Release lock
						if err := locker.SessionUnlock(ctx, conn); err != nil {
							t.Errorf("worker %d: failed to unlock: %v", id, err)
						}

						// Stop after successful lock/unlock
						return
					}
					// If we didn't get the lock, retry after a short delay
					time.Sleep(10 * time.Millisecond)
				}
			}(i)
		}

		wg.Wait()

		// All workers should have successfully acquired the lock at some point
		require.Equal(t, int64(numWorkers), successfulLocks, "all workers should eventually get the lock")
	})
}

// TestDialectConcurrentHeartbeat tests heartbeat functionality under concurrent load
func TestDialectConcurrentHeartbeat(t *testing.T) {
	dialects := []struct {
		name    string
		dialect database.Dialect
		dsn     func() string
	}{
		{
			name:    "postgres",
			dialect: database.DialectPostgres,
			dsn:     func() string { return testPostgresDB(t) },
		},
		{
			name:    "sqlite",
			dialect: database.DialectSQLite3,
			dsn:     func() string { return testSQLiteDB(t) },
		},
	}

	for _, d := range dialects {
		t.Run(d.name, func(t *testing.T) {
			if d.name == "postgres" && !isPostgresAvailable() {
				t.Skip("PostgreSQL not available")
			}

			testDialectConcurrentHeartbeatImpl(t, d.dialect, d.dsn())
		})
	}
}

func testDialectConcurrentHeartbeatImpl(t *testing.T, dialect database.Dialect, dsn string) {
	db, err := sql.Open(getDriverName(dialect), dsn)
	require.NoError(t, err)
	defer db.Close()

	// Create locker with fast heartbeat for testing
	locker, err := NewTableSessionLockerForDialect(
		dialect,
		WithHeartbeatInterval(1*time.Second), // Minimum allowed is 1 second
		WithStaleTimeout(time.Minute),       // Minimum allowed is 1 minute
		WithLockTimeout(1, 40),              // 1 second intervals, 40 attempts
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn.Close()

	// Acquire lock
	err = locker.SessionLock(ctx, conn)
	require.NoError(t, err)

	// Launch multiple goroutines trying to compete for the lock
	// while the current holder maintains heartbeat
	var wg sync.WaitGroup
	numCompetitors := 8

	for i := 0; i < numCompetitors; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			competitorConn, err := db.Conn(ctx)
			if err != nil {
				t.Errorf("competitor %d: failed to get connection: %v", id, err)
				return
			}
			defer competitorConn.Close()

			// This should fail because the lock is held and heartbeat is active
			err = locker.SessionLock(ctx, competitorConn)
			if err == nil {
				t.Errorf("competitor %d: should not have acquired lock while heartbeat is active", id)
				locker.SessionUnlock(ctx, competitorConn)
			}
		}(i)
	}

	// Let heartbeat run while competitors try
	time.Sleep(2 * time.Second)

	// Release the lock
	err = locker.SessionUnlock(ctx, conn)
	require.NoError(t, err)

	wg.Wait()
}

// TestDialectStaleLocksWithConcurrency tests stale lock detection and cleanup under concurrent load
func TestDialectStaleLocksWithConcurrency(t *testing.T) {
	dialects := []struct {
		name    string
		dialect database.Dialect
		dsn     func() string
	}{
		{
			name:    "postgres",
			dialect: database.DialectPostgres,
			dsn:     func() string { return testPostgresDB(t) },
		},
		{
			name:    "sqlite",
			dialect: database.DialectSQLite3,
			dsn:     func() string { return testSQLiteDB(t) },
		},
	}

	for _, d := range dialects {
		t.Run(d.name, func(t *testing.T) {
			if d.name == "postgres" && !isPostgresAvailable() {
				t.Skip("PostgreSQL not available")
			}

			testDialectStaleLocksWithConcurrencyImpl(t, d.dialect, d.dsn())
		})
	}
}

func testDialectStaleLocksWithConcurrencyImpl(t *testing.T, dialect database.Dialect, dsn string) {
	db, err := sql.Open(getDriverName(dialect), dsn)
	require.NoError(t, err)
	defer db.Close()

	// Configure SQLite for proper concurrency  
	if dialect == database.DialectSQLite3 {
		_, err = db.Exec("PRAGMA journal_mode=WAL; PRAGMA locking_mode=NORMAL; PRAGMA synchronous=NORMAL;")
		require.NoError(t, err)
	}

	ctx := context.Background()

	// Create first locker using same pattern as working table tests
	var locker1 SessionLocker
	if dialect == database.DialectSQLite3 {
		store := NewLockStore(DefaultLockTableName, NewSQLiteLockQuerier())
		locker1, err = NewTableSessionLockerWithStore(store, 
			WithStaleTimeout(time.Minute),
			WithHeartbeatInterval(time.Second),
		)
	} else {
		locker1, err = NewTableSessionLockerForDialect(
			dialect,
			WithStaleTimeout(time.Minute),
			WithHeartbeatInterval(time.Second),
		)
	}
	require.NoError(t, err)

	// First connection acquires lock
	conn1, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn1.Close()

	err = locker1.SessionLock(ctx, conn1)
	require.NoError(t, err)

	// Simulate stale lock by manually updating the heartbeat to be very old
	err = makeStale(ctx, conn1)
	require.NoError(t, err)

	// Create second locker that should clean up stale lock
	var locker2 SessionLocker
	if dialect == database.DialectSQLite3 {
		store := NewLockStore(DefaultLockTableName, NewSQLiteLockQuerier())
		locker2, err = NewTableSessionLockerWithStore(store, 
			WithStaleTimeout(time.Minute),
			WithLockTimeout(1, 3), // Quick timeout
		)
	} else {
		locker2, err = NewTableSessionLockerForDialect(
			dialect,
			WithStaleTimeout(time.Minute),
			WithLockTimeout(1, 3), // Quick timeout
		)
	}
	require.NoError(t, err)

	conn2, err := db.Conn(ctx)
	require.NoError(t, err)
	defer conn2.Close()

	// Second locker should clean up stale lock and acquire it
	err = locker2.SessionLock(ctx, conn2)
	require.NoError(t, err)

	// Clean up
	err = locker2.SessionUnlock(ctx, conn2)
	require.NoError(t, err)
}

// TestDialectHighConcurrencyStress performs stress testing with many concurrent operations
func TestDialectHighConcurrencyStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	dialects := []struct {
		name    string
		dialect database.Dialect
		dsn     func() string
	}{
		{
			name:    "postgres",
			dialect: database.DialectPostgres,
			dsn:     func() string { return testPostgresDB(t) },
		},
		{
			name:    "sqlite",
			dialect: database.DialectSQLite3,
			dsn:     func() string { return testSQLiteDB(t) },
		},
	}

	for _, d := range dialects {
		t.Run(d.name, func(t *testing.T) {
			if d.name == "postgres" && !isPostgresAvailable() {
				t.Skip("PostgreSQL not available")
			}

			testDialectHighConcurrencyStressImpl(t, d.dialect, d.dsn())
		})
	}
}

func testDialectHighConcurrencyStressImpl(t *testing.T, dialect database.Dialect, dsn string) {
	db, err := sql.Open(getDriverName(dialect), dsn)
	require.NoError(t, err)
	defer db.Close()

	// Set connection limits for stress testing
	db.SetMaxOpenConns(50)
	db.SetMaxIdleConns(10)

	locker, err := NewTableSessionLockerForDialect(
		dialect,
		WithHeartbeatInterval(1*time.Second), // Minimum allowed is 1 second
		WithStaleTimeout(time.Minute),        // Minimum allowed is 1 minute
		WithLockTimeout(1, 30),               // 1 second intervals, 30 attempts
	)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var totalAttempts int64
	var successfulLocks int64
	var mu sync.Mutex
	var wg sync.WaitGroup

	// High concurrency stress test
	numWorkers := runtime.GOMAXPROCS(0) * 8
	if numWorkers < 20 {
		numWorkers = 20
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				conn, err := db.Conn(ctx)
				if err != nil {
					continue // Connection pool exhausted, retry
				}

				mu.Lock()
				totalAttempts++
				mu.Unlock()

				// Try to acquire lock
				if err := locker.SessionLock(ctx, conn); err == nil {
					mu.Lock()
					successfulLocks++
					mu.Unlock()

					// Simulate very brief work
					time.Sleep(5 * time.Millisecond)

					locker.SessionUnlock(ctx, conn)
				}

				conn.Close()

				// Brief pause between attempts to reduce contention
				time.Sleep(20 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Verify we had significant activity and some successful locks
	require.Greater(t, totalAttempts, int64(50), "should have many lock attempts")
	require.Greater(t, successfulLocks, int64(1), "should have some successful locks")

	t.Logf("Stress test completed: %d total attempts, %d successful locks (%.2f%% success rate)",
		totalAttempts, successfulLocks, float64(successfulLocks)/float64(totalAttempts)*100)
}

// Helper functions
func getDriverName(dialect database.Dialect) string {
	switch dialect {
	case database.DialectPostgres:
		return "postgres"
	case database.DialectSQLite3:
		return "sqlite"
	default:
		return "unknown"
	}
}

func testSQLiteDB(t *testing.T) string {
	// Use a temporary file database for proper concurrency testing like existing table tests
	tmpfile := t.TempDir() + "/test_concurrent.db"
	return tmpfile
}

func testPostgresDB(_ *testing.T) string {
	// This would need to be configured for your test environment
	// For now, return a test DSN - adjust as needed for your setup
	return "postgres://user:password@localhost/testdb?sslmode=disable"
}

func isPostgresAvailable() bool {
	// Quick check if PostgreSQL is available for testing
	db, err := sql.Open("postgres", "postgres://user:password@localhost/testdb?sslmode=disable")
	if err != nil {
		return false
	}
	defer db.Close()
	return db.Ping() == nil
}

