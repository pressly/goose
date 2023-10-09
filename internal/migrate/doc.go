// Package migrate defines a Migration struct and implements the migration logic for executing Go
// and SQL migrations.
//
//   - For Go migrations, only *sql.Tx and *sql.DB are supported. *sql.Conn is not supported.
//   - For SQL migrations, all three are supported.
//
// Lastly, SQL migrations are lazily parsed. This means that the SQL migration is parsed the first
// time it is executed.
package migrate
