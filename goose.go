package goose

import (
	"database/sql"
	"fmt"
	"sync"
)

var (
	duplicateCheckOnce sync.Once
	minVersion         = int64(0)
	maxVersion         = int64((1 << 63) - 1)
)

func Run(command string, db *sql.DB, dir string, args ...string) error {
	switch command {
	case "up":
		if err := Up(db, dir); err != nil {
			return err
		}
	case "up-by-one":
		if err := UpByOne(db, dir); err != nil {
			return err
		}
	case "create":
		if len(args) == 0 {
			return fmt.Errorf("create must be of form: goose [OPTIONS] DRIVER DBSTRING create NAME [go|sql]")
		}

		migrationType := "go"
		if len(args) == 2 {
			migrationType = args[1]
		}
		if err := Create(db, dir, args[0], migrationType); err != nil {
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
