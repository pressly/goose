package locktesting

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3/lock"
)

// TestConcurrentSessionLocking verifies the locking contract of a custom
// [lock.SessionLocker]. It checks exclusive acquisition under contention,
// unlocking on the original connection, and subsequent reacquisition on a
// different connection.
//
// newLocker must return a new locker competing for the SAME lock resource on
// every call. The database pool must allow at least five simultaneous
// connections, because session locks belong to individual connections.
// lockTimeout must be positive. Each contender gets this long to acquire the
// lock; its cancellation must not prevent the winner from unlocking.
func TestConcurrentSessionLocking(
	t *testing.T,
	db *sql.DB,
	newLocker func(*testing.T) lock.SessionLocker,
	lockTimeout time.Duration,
) {
	t.Helper()
	if db == nil || newLocker == nil || lockTimeout <= 0 {
		t.Fatal("db and newLocker must be non-nil and lockTimeout must be positive")
	}

	const count = 5
	lockers := make([]lock.SessionLocker, count)
	conns := make([]*sql.Conn, count)
	for i := range lockers {
		lockers[i] = newLocker(t)
		if lockers[i] == nil {
			t.Fatalf("newLocker returned nil for contender %d", i)
		}

		ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
		conn, err := db.Conn(ctx)
		cancel()
		if err != nil {
			t.Fatalf("acquire connection %d: %v (the pool must permit %d connections)", i, err, count)
		}
		conns[i] = conn
		index := i
		t.Cleanup(func() {
			if conns[index] != nil {
				if err := conns[index].Close(); err != nil {
					t.Errorf("close session connection %d: %v", index, err)
				}
			}
		})
	}

	type result struct {
		index int
		err   error
	}
	attempts := make(chan result, count)
	unlocks := make(chan result, count)
	start := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	releaseWinner := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseWinner()

	var wg sync.WaitGroup
	wg.Add(count)
	for i := range lockers {
		go func(i int) {
			defer wg.Done()
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), lockTimeout)
			err := lockers[i].SessionLock(ctx, conns[i])
			cancel()
			attempts <- result{index: i, err: err}
			if err != nil {
				return
			}

			// Keep the successful session lock held until every contender has
			// returned. This makes overlapping acquisitions observable without
			// sleeps and keeps the unlock context independent of lockTimeout.
			<-release
			unlockCtx, unlockCancel := context.WithTimeout(context.Background(), lockTimeout)
			defer unlockCancel()
			unlocks <- result{index: i, err: lockers[i].SessionUnlock(unlockCtx, conns[i])}
		}(i)
	}
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	close(start)

	timer := time.NewTimer(lockTimeout + 5*time.Second)
	defer timer.Stop()
	winners := make([]int, 0, 1)
	for range count {
		select {
		case attempt := <-attempts:
			if attempt.err == nil {
				winners = append(winners, attempt.index)
			}
		case <-timer.C:
			t.Fatalf("session lock attempts did not finish within %v; check context handling and connection pool size", lockTimeout+5*time.Second)
		}
	}

	releaseWinner()
	select {
	case <-done:
	case <-time.After(lockTimeout + 5*time.Second):
		t.Fatal("session unlock did not finish; check whether the locker respects its context")
	}

	close(unlocks)
	for unlock := range unlocks {
		if unlock.err != nil {
			t.Errorf("locker %d failed to unlock using the same session: %v", unlock.index, unlock.err)
		}
	}
	if len(winners) != 1 {
		t.Fatalf("expected exactly one successful session lock, got %d (winners: %v)", len(winners), winners)
	}

	// An unlock must actually release the resource, not merely return nil.
	// Use a fresh connection distinct from the winner. Replacing a losing
	// connection also avoids depending on driver-specific behavior after a
	// canceled lock attempt, without requiring a sixth pool slot.
	next := (winners[0] + 1) % count
	if err := conns[next].Close(); err != nil {
		t.Fatalf("close losing connection %d: %v", next, err)
	}
	conns[next] = nil
	retryCtx, retryCancel := context.WithTimeout(context.Background(), lockTimeout)
	newConn, err := db.Conn(retryCtx)
	if err != nil {
		retryCancel()
		t.Fatalf("acquire fresh session after unlock: %v", err)
	}
	conns[next] = newConn
	err = lockers[next].SessionLock(retryCtx, newConn)
	retryCancel()
	if err != nil {
		t.Fatalf("locker %d could not reacquire after unlock: %v", next, err)
	}
	unlockCtx, unlockCancel := context.WithTimeout(context.Background(), lockTimeout)
	defer unlockCancel()
	if err := lockers[next].SessionUnlock(unlockCtx, newConn); err != nil {
		t.Errorf("locker %d could not release reacquired lock: %v", next, err)
	}
}
