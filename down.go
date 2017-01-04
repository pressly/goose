package goose

import (
	"database/sql"
	"fmt"
)

func (c *Client) Down(db *sql.DB, dir string) error {
	currentVersion, err := c.GetDBVersion(db)
	if err != nil {
		return err
	}

	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("no migration %v", currentVersion)
	}

	return c.runMigration(db, current, migrateDown)
}
