package postgres

// type postgres struct {
// 	db *sql.DB
// }

// var _ dialect.SQLDialect = (*postgres)(nil)

// func New(db *sql.DB) (dialect.SQLDialect, error) {
// 	return &postgres{
// 		db: db,
// 	}, nil
// }

// var createTableSQL = `CREATE TABLE %s (
//     id serial NOT NULL,
//     version_id bigint NOT NULL,
//     is_applied boolean NOT NULL,
//     tstamp timestamp NULL default now(),
//     PRIMARY KEY(id)`

// func (p *postgres) CreateTable(tableName string) error {
// 	return fmt.Sprintf(createTableSQL, tableName), nil
// }

// var insertVersionSQL = `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`

// func (p *postgres) insertVersionSQL(tableName string, versionID int) string {
// 	return fmt.Sprintf(insertVersionSQL, versionID)
// }
