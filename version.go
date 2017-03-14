package goose

import (
	"database/sql"
	"fmt"
)

func (c *Client) Version(db *sql.DB, dir string) error {
	current, err := c.GetDBVersion(db)
	if err != nil {
		return err
	}

	fmt.Printf("goose: dbversion %v\n", current)
	return nil
}
