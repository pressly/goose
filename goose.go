package goose

import (
	"database/sql"
	"fmt"
)

func Run(command string, db *sql.DB, dir string) error {
	switch command {
	case "up":
		if err := Up(db, dir); err != nil {
			return err
		}
	case "down":
		if err := Down(db, dir); err != nil {
			return err
		}
	case "redo":
		if err := Redo(db, dir); err != nil {
			return err
		}
	case "status":
		if err := Status(db, dir); err != nil {
			return err
		}
	case "version":
		if err := Version(db, dir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%q: no such command", command)
	}
	return nil
}
