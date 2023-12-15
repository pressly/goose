package gomigrations

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(up002, nil)
}

func up002(db *sql.DB) error {
	for i := 1; i <= 100; i++ {
		q := "INSERT INTO foo VALUES ($1)"
		if _, err := db.Exec(q, i); err != nil {
			return err
		}
		// Simulate an error when no tx. We should have 50 rows
		// inserted in the DB.
		if i == 50 {
			return fmt.Errorf("simulate error: too many inserts")
		}
	}
	return nil
}
