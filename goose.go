package goose

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync"
)

type RunParams struct {
	Dir            *string
	MissingOnly    *bool
	IncludeMissing *bool
}

var (
	duplicateCheckOnce sync.Once
	minVersion         = int64(0)
	maxVersion         = int64((1 << 63) - 1)
	timestampFormat    = "20060102150405"
)

// Run runs a goose command.
func Run(command string, db *sql.DB, params RunParams, args ...string) error {
	switch command {
	case "up":
		if err := Up(db, *params.Dir, *params.IncludeMissing, false, nil); err != nil {
			return err
		}
	case "up-by-one":
		if err := Up(db, *params.Dir, *params.IncludeMissing, true, nil); err != nil {
			return err
		}
	case "up-to":
		if len(args) == 0 {
			return fmt.Errorf("up-to must be of form: goose [OPTIONS] DRIVER DBSTRING up-to VERSION")
		}

		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("version must be a number (got '%s')", args[0])
		}
		if err := Up(db, *params.Dir, *params.IncludeMissing, false, &version); err != nil {
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
		if err := Create(db, *params.Dir, args[0], migrationType); err != nil {
			return err
		}
	case "down":
		if err := Down(db, *params.Dir); err != nil {
			return err
		}
	case "down-to":
		if len(args) == 0 {
			return fmt.Errorf("down-to must be of form: goose [OPTIONS] DRIVER DBSTRING down-to VERSION")
		}

		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("version must be a number (got '%s')", args[0])
		}
		if err := DownTo(db, *params.Dir, version); err != nil {
			return err
		}
	case "fix":
		if err := Fix(*params.Dir); err != nil {
			return err
		}
	case "redo":
		if err := Redo(db, *params.Dir); err != nil {
			return err
		}
	case "reset":
		if err := Reset(db, *params.Dir); err != nil {
			return err
		}
	case "status":
		if *params.MissingOnly {
			if err := StatusMissing(db, *params.Dir); err != nil {
				return err
			}
		} else {
			if err := Status(db, *params.Dir); err != nil {
				return err
			}
		}
	case "version":
		if err := Version(db, *params.Dir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%q: no such command", command)
	}
	return nil
}
