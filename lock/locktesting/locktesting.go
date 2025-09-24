package locktesting

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

// TestProviderLocking is a reusable test helper that verifies locking behavior. It creates the a
// bunch of providers with the same locker configuration and verifies that only one provider can
// apply migrations at a time.
//
// The test verifies the core locking contract:
//
//  1. Only one provider should apply migrations when running concurrently
//  2. The other providers should apply zero migrations (blocked by lock)
//  3. All providers should complete without errors
//  4. Final database state should be consistent
func TestProviderLocking(
	t *testing.T,
	newProvider func(*testing.T) *goose.Provider,
) {
	t.Helper()

	// Number of concurrent providers to test
	const count = 5

	// Create providers
	providers := make([]*goose.Provider, count)
	for i := range count {
		providers[i] = newProvider(t)
	}
	// Sanity check - ensure providers have migration sources
	sources := providers[0].ListSources()
	require.NotEmpty(t, sources, "no migration sources found - check provider fsys")
	maxVersion := sources[len(sources)-1].Version
	// Ensure all providers have the same sources
	for _, p := range providers {
		require.Equal(t, sources, p.ListSources(), "providers have different migration sources")
	}

	// Since locking is enabled, only one of these providers should apply ALL the migrations. The
	// other providers should apply NO migrations.

	var g errgroup.Group
	results := make([]int, count)

	for i := range count {
		g.Go(func() error {
			ctx := context.Background()
			migrationResults, err := providers[i].Up(ctx)
			if err != nil {
				return err
			}
			results[i] = len(migrationResults)
			// Useful for debugging:
			//
			// t.Logf("Provider %d applied %d migrations", i, len(migrationResults))
			currentVersion, err := providers[i].GetDBVersion(ctx)
			if err != nil {
				return err
			}
			if currentVersion != maxVersion {
				return fmt.Errorf("provider %d: expected version %d, got %d", i, maxVersion, currentVersion)
			}
			return nil
		})
	}
	require.NoError(t, g.Wait())

	// Verify locking behavior: exactly one provider should have done all the work
	var (
		providersWithWork   = 0
		providerWithAllWork = -1 // Index of provider that did all the work
	)
	for i, res := range results {
		if res > 0 {
			providersWithWork++
			if res == len(sources) {
				providerWithAllWork = i
			}
		}
	}
	// Verify exactly one provider did work
	require.Equal(t, 1, providersWithWork, "exactly one provider should apply migrations - locking is not working")
	// Verify that provider did all the work
	require.NotEqual(t, -1, providerWithAllWork, "one provider should have applied all migrations - locking is not working")
	// Verify all others did no work
	for i, res := range results {
		if i != providerWithAllWork {
			require.Equal(t, 0, res, "provider%d should have applied 0 migrations", i)
		}
	}
}

// TestConcurrentLocking is a reusable test helper that verifies concurrent locker behavior. It
// creates the specified number of lockers using the factory function and verifies that only one
// locker can acquire the lock at a time.
//
// IMPORTANT: The newLocker function MUST create lockers that compete for the SAME lock resource.
// For table-based lockers, this means using the same lock ID. For advisory locks, the same lock ID.
// If each locker targets a different resource, multiple lockers will succeed (which breaks the
// test).
//
// The test verifies the core locking contract:
//
//  1. Only one locker should successfully acquire the lock when running concurrently
//  2. The other lockers should fail to acquire the lock (blocked/timeout)
//  3. All lockers should complete without hanging
func TestConcurrentLocking(
	t *testing.T,
	db *sql.DB,
	newLocker func(*testing.T) lock.Locker,
	lockTimeout time.Duration,
) {
	t.Helper()
	ctx := context.Background()

	// TODO(mf): I wonder if there's a better way to do logging in tests that conditionally enables
	// it. Maybe using testing.T.Log? But that doesn't have levels. Maybe use a global flag to
	// enable debug logging in tests?

	// logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	// logger := slog.New(slog.DiscardHandler)

	// Number of concurrent lockers to test
	const count = 5

	lockers := make([]lock.Locker, count)
	for i := range count {
		lockers[i] = newLocker(t)
	}

	// Use buffered channel to collect successful lock acquisitions
	successCh := make(chan int, count)
	var wg sync.WaitGroup

	// Start multiple goroutines trying to acquire the same lock
	for i := range count {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(ctx, lockTimeout)
			defer cancel()

			// Try to acquire the lock
			if err := lockers[i].Lock(ctx, db); err != nil {
				// logger.Debug("Locker failed to acquire lock", slog.Int("locker", i), slog.String("error", err.Error()))
				return
			}

			successCh <- i
			// logger.Debug("Locker acquired lock", slog.Int("locker", i))

			// Hold the lock long enough for all other goroutines to exhaust their retries. This
			// ensures only ONE locker succeeds in the concurrent test
			time.Sleep(lockTimeout * 2)

			// Release the lock
			if err := lockers[i].Unlock(ctx, db); err != nil {
				t.Errorf("Locker %d failed to release lock: %v", i, err)
			}
			// } else {
			// logger.Debug("Locker released lock", slog.Int("locker", i))
			// }
		}()
	}
	// Wait for all goroutines with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Test completed normally
	case <-time.After(lockTimeout + 5*time.Second):
		t.Fatal("Test timed out - lockers took too long")
	}

	// Collect results from channel
	close(successCh)
	var successful []int
	for id := range successCh {
		successful = append(successful, id)
	}

	require.Len(t, successful, 1, "Exactly one locker should acquire the lock")
	// logger.Debug("Concurrent locking test passed", slog.Int("winning_locker", successful[0]))
}
