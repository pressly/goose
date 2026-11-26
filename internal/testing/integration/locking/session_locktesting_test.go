package locking_test

import (
	"testing"
	"time"

	"github.com/pressly/goose/v3/internal/testing/testdb"
	"github.com/pressly/goose/v3/lock"
	"github.com/pressly/goose/v3/lock/locktesting"
)

func TestPostgresSessionLockerContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping PostgreSQL integration test in short mode")
	}

	db, cleanup, err := testdb.NewPostgres()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanup)
	db.SetMaxOpenConns(5)

	locktesting.TestConcurrentSessionLocking(t, db, func(t *testing.T) lock.SessionLocker {
		locker, err := lock.NewPostgresSessionLocker(
			lock.WithLockID(991),
			lock.WithLockTimeout(1, 2),
			lock.WithUnlockTimeout(1, 2),
		)
		if err != nil {
			t.Fatal(err)
		}
		return locker
	}, 4*time.Second)
}
