package locktesting

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3/lock"
)

// This driver supplies distinct *sql.Conn handles without requiring a real DB.
// The fake locker below exercises the helper's concurrency protocol.
type sessionTestDriver struct{}

type sessionTestConn struct{}

func (sessionTestDriver) Open(string) (driver.Conn, error) { return sessionTestConn{}, nil }
func (sessionTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("not used")
}
func (sessionTestConn) Close() error              { return nil }
func (sessionTestConn) Begin() (driver.Tx, error) { return nil, errors.New("not used") }

var registerSessionTestDriver sync.Once

var _ lock.SessionLocker = (*testSessionLocker)(nil)

type testSessionLocker struct {
	available chan struct{}
	mu        sync.Mutex
	owner     *sql.Conn
}

func (l *testSessionLocker) SessionLock(ctx context.Context, conn *sql.Conn) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-l.available:
		l.mu.Lock()
		l.owner = conn
		l.mu.Unlock()
		return nil
	}
}

func (l *testSessionLocker) SessionUnlock(ctx context.Context, conn *sql.Conn) error {
	l.mu.Lock()
	if l.owner != conn {
		l.mu.Unlock()
		return errors.New("not the connection that acquired the lock")
	}
	l.owner = nil
	l.mu.Unlock()
	select {
	case l.available <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func TestConcurrentSessionLockingHelper(t *testing.T) {
	registerSessionTestDriver.Do(func() {
		sql.Register("goose_locktesting_session_stub", sessionTestDriver{})
	})
	db, err := sql.Open("goose_locktesting_session_stub", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(5)

	available := make(chan struct{}, 1)
	available <- struct{}{}
	newLocker := func(*testing.T) lock.SessionLocker {
		return &testSessionLocker{available: available}
	}
	TestConcurrentSessionLocking(t, db, newLocker, 100*time.Millisecond)
}
