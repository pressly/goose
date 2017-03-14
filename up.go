package goose

import (
	"database/sql"
	"fmt"
)

func (c *Client) Up(db *sql.DB, dir string) error {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	for {
		current, err := c.GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == ErrNoNextVersion {
				fmt.Printf("goose: no migrations to run. current version: %d\n", current)
				return nil
			}
			return err
		}

		if err = c.runMigration(db, next, migrateUp); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) UpByOne(db *sql.DB, dir string) error {
	migrations, err := c.collectMigrations(dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	currentVersion, err := c.GetDBVersion(db)
	if err != nil {
		return err
	}

	next, err := migrations.Next(currentVersion)
	if err != nil {
		if err == ErrNoNextVersion {
			fmt.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
		}
		return err
	}

	if err = c.runMigration(db, next, migrateUp); err != nil {
		return err
	}

	return nil
}
