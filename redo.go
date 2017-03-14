package goose

import (
	"database/sql"
)

func (c *Client) Redo(db *sql.DB, dir string) error {
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
		return err
	}

	previous, err := migrations.Next(currentVersion)
	if err != nil {
		return err
	}

	if err := c.runMigration(db, previous, migrateUp); err != nil {
		return err
	}

	if err := c.runMigration(db, current, migrateUp); err != nil {
		return err
	}

	return nil
}
