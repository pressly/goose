package lock

// LockQuerier defines the interface for database-specific lock SQL generation.
// This follows the same pattern as database/dialect.Querier for consistency.
type LockQuerier interface {
	// CreateLockTable returns the SQL query string to create the lock table.
	CreateLockTable(tableName string) string
	
	// AcquireLock returns the SQL query string to atomically acquire a lock.
	// This query should use an UPDATE with WHERE to ensure exactly one process wins.
	// Parameters: processInfo (string identifying the lock holder)
	AcquireLock(tableName string) string
	
	// InsertInitialLock returns the SQL query string to insert the initial lock row.
	// This is used when the table is empty and needs the first lock row.
	// Parameters: processInfo (string identifying the lock holder)
	InsertInitialLock(tableName string) string
	
	// ReleaseLock returns the SQL query string to release a held lock.
	ReleaseLock(tableName string) string
	
	// UpdateHeartbeat returns the SQL query string to update the heartbeat timestamp.
	UpdateHeartbeat(tableName string) string
	
	// CleanupStaleLocks returns the SQL query string to clean up stale locks.
	// Parameters: staleTimeoutSeconds (int - seconds after which a lock is considered stale)
	CleanupStaleLocks(tableName string) string
}