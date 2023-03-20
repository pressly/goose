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

func NewPostgres(table string) Querier {
	return &postgres{table: table}
}

func NewMysql(table string) Querier {
	return &mysql{table: table}
}

func NewSqlite3(table string) Querier {
	return &sqlite3{table: table}
}

func NewSqlserver(table string) Querier {
	return &sqlserver{table: table}
}

func NewRedshift(table string) Querier {
	return &redshift{table: table}
}

func NewTidb(table string) Querier {
	return &tidb{table: table}
}

func NewClickhouse(table string) Querier {
	return &clickhouse{table: table}
}

func NewVertica(table string) Querier {
	return &vertica{table: table}
}
