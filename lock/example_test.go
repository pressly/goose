package lock_test

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/lock"
	_ "modernc.org/sqlite"
)

func ExampleNewTableSessionLocker() {
	// Open a database connection
	db, err := sql.Open("sqlite", "example.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create a table-based session locker optimized for your database
	// Supported dialects: database.DialectPostgres, database.DialectSQLite3
	// For other databases, use lock.NewTableSessionLocker() for generic SQL
	locker, err := lock.NewTableSessionLockerForDialect(
		database.DialectSQLite3, // Use database.DialectPostgres for PostgreSQL
		lock.WithHeartbeatInterval(30*time.Second), // Heartbeat every 30 seconds
		lock.WithStaleTimeout(5*time.Minute),       // Consider locks stale after 5 minutes
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Acquire the lock
	if err := locker.SessionLock(ctx, conn); err != nil {
		log.Fatal("Failed to acquire lock:", err)
	}
	
	// Perform migrations or other critical operations here
	log.Println("Lock acquired, performing operations...")
	
	// Release the lock when done
	if err := locker.SessionUnlock(ctx, conn); err != nil {
		log.Fatal("Failed to release lock:", err)
	}
	
	log.Println("Lock released")
}