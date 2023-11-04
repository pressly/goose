package gomigrations

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
)

func init() {
	goose.AddMigration(up001, down001)
}

func up001(tx *sql.Tx) error {
	return createTable(tx, "alpha")
}

func down001(tx *sql.Tx) error {
	return dropTable(tx, "alpha")
}

func createTable(db database.DBTxConn, name string) error {
	_, err := db.ExecContext(context.Background(), fmt.Sprintf("CREATE TABLE %s (id INTEGER)", name))
	return err
}

func dropTable(db database.DBTxConn, name string) error {
	_, err := db.ExecContext(context.Background(), fmt.Sprintf("DROP TABLE %s", name))
	return err
}
