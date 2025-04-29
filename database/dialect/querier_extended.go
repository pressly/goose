package dialect

// QuerierExtender is an extension of the Querier interface that provides optional optimizations and
// database-specific features. While not required by the core package, implementing these methods
// can improve performance and functionality for specific databases.
//
// IMPORTANT: This interface may be expanded in future versions. Implementors MUST be prepared to
// update their implementations when new methods are added, either by implementing the new
// functionality or returning an empty string.
//
// Example usage to verify implementation:
//
//	var _ QuerierExtender = (*CustomQuerierExtended)(nil)
//
// In short, it's exported to allows implementors to have a compile-time check that they are
// implementing the interface correctly.
type QuerierExtender interface {
	Querier

	// TableExists returns the SQL query string to check if a table exists in the database.
	// Implementing this method allows the system to optimize table existence checks by using
	// database-specific system catalogs (e.g., pg_tables for PostgreSQL, sqlite_master for SQLite)
	// instead of generic SQL queries.
	//
	// Return an empty string if the database does not provide an efficient way to check table
	// existence.
	TableExists(tableName string) string
}
