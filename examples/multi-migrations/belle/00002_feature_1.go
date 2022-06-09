package belle

import (
	"database/sql"
)

func init() {
	Provider.AddMigration(upFeature1, downFeature1)

}

func upFeature1(tx *sql.Tx) error {
	// This code is executed when the migration is applied.
	return nil
}

func downFeature1(tx *sql.Tx) error {
	// This code is executed when the migration is rolled back.
	return nil
}
