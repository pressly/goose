package dialect

/*

	Working notes.

	is_applied serves no purpose, but it's also got a NOT NULL constraint so we need to
	ensure this continues to work.

	for context, this used to be this way because the way down migrations worked.
	but now we do explicit deletions.
	https://github.com/pressly/goose/pull/131#pullrequestreview-178409168

*/

type SQL interface {
	// CreateTable defines the SQL query for creating a version table to
	// store goose migrations.
	//
	// This should be set on the underlying implementation and used for
	// all other queries.
	CreateTable() string

	// InsertVersion defines the SQL query for inserting a version id.
	InsertVersion(version int64) string

	// DeleteVersion defines the SQL query for deleting a version id.
	DeleteVersion(version int64) string

	// GetMigration defines the SQL query to get one migration by version id.
	// TODO(mf): this is inefficient. Only used in one place to list migrations one-by-one
	// but we can do better.
	// Oh, and selecting by version id does not have an index ..
	GetMigration(version int64) string

	// ListMigrations defines the SQL query to list all migrations in
	// ascending order by id.
	ListMigrations() string
}
