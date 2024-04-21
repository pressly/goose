package dialectquery

// Querier is the interface that wraps the basic methods to create a dialect specific query.
type Querier interface {
	// CreateTable returns the SQL query string to create the db version table.
	CreateTable(tableName string) string

	// InsertVersion returns the SQL query string to insert a new version into the db version table.
	InsertVersion(tableName string) string

	// DeleteVersion returns the SQL query string to delete a version from the db version table.
	DeleteVersion(tableName string) string

	// GetMigrationByVersion returns the SQL query string to get a single migration by version.
	//
	// The query should return the timestamp and is_applied columns.
	GetMigrationByVersion(tableName string) string

	// ListMigrations returns the SQL query string to list all migrations in descending order by id.
	//
	// The query should return the version_id and is_applied columns.
	ListMigrations(tableName string) string
}

type QueryController struct {
	querier Querier
}

// NewQueryController returns a new QueryController that wraps the given Querier.
func NewQueryController(querier Querier) *QueryController {
	return &QueryController{querier: querier}
}

// Optional methods

// TableExists returns the SQL query string to check if the version table exists. If the Querier
// does not implement this method, it will return an empty string.
//
// The query should return a boolean value.
func (c *QueryController) TableExists(tableName string) string {
	if t, ok := c.querier.(interface {
		TableExists(string) string
	}); ok {
		return t.TableExists(tableName)
	}
	return ""
}

// Default methods

func (c *QueryController) CreateTable(tableName string) string {
	return c.querier.CreateTable(tableName)
}

func (c *QueryController) InsertVersion(tableName string) string {
	return c.querier.InsertVersion(tableName)
}

func (c *QueryController) DeleteVersion(tableName string) string {
	return c.querier.DeleteVersion(tableName)
}

func (c *QueryController) GetMigrationByVersion(tableName string) string {
	return c.querier.GetMigrationByVersion(tableName)
}

func (c *QueryController) ListMigrations(tableName string) string {
	return c.querier.ListMigrations(tableName)
}
