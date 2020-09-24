package goose

// Version prints the current version of the database.
func Version(cfg *config) error {
	current, err := GetDBVersion(cfg.db)
	if err != nil {
		return err
	}

	log.Printf("goose: version %v\n", current)
	return nil
}

var tableName = "goose_db_version"

// TableName returns goose db version table name
func TableName() string {
	return tableName
}

// SetTableName set goose db version table name
func SetTableName(n string) {
	tableName = n
}
