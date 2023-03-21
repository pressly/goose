package dialectquery

// Querier is the interface that wraps the basic methods to create a dialect
// specific query.
type Querier interface {
	// CreateTable returns the SQL query string to create the db version table.
	CreateTable() string

	// InsertVersion returns the SQL query string to insert a new version into
	// the db version table.
	InsertVersion() string

	// DeleteVersion returns the SQL query string to delete a version from
	// the db version table.
	DeleteVersion() string

	// GetMigrationByVersion returns the SQL query string to get a single
	// migration by version.
	//
	// The query should return the timestamp and is_applied columns.
	GetMigrationByVersion() string

	// ListMigrations returns the SQL query string to list all migrations in
	// descending order by id.
	//
	// The query should return the version_id and is_applied columns.
	ListMigrations() string
}
