package gomigrations

import (
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(up004, nil)
}

func up004(tx *sql.Tx) error {
	for i := 1; i <= 100; i++ {
		// Simulate an error when no tx. We should have 50 rows
		// inserted in the DB.
		if i == 50 {
			return fmt.Errorf("simulate error: too many inserts")
		}
		q := "INSERT INTO foo VALUES ($1)"
		if _, err := tx.Exec(q); err != nil {
			return err
		}
	}
	return nil
}
