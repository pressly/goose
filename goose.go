package goose

import (
	"database/sql"
	"fmt"
	"strconv"
)

const VERSION = "v2.7.0-rc3"

var (
	minVersion         = int64(0)
	maxVersion         = int64((1 << 63) - 1)
	timestampFormat    = "20060102150405"
	verbose            = false
)

// SetVerbose set the goose verbosity mode
func SetVerbose(v bool) {
	verbose = v
}

// Run runs a goose command.
func Run(command string, db *sql.DB, dir string, args ...string) error {
	opts := newConfig(dir, db)
	switch command {
	case "up":
		if err := Up(opts); err != nil {
			return err
		}
	case "up-by-one":
		if err := UpByOne(opts); err != nil {
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
		if err := UpTo(opts, version); err != nil {
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
		if err := Create(opts, args[0], migrationType); err != nil {
			return err
		}
	case "down":
		if err := Down(opts); err != nil {
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
		if err := DownTo(opts, version); err != nil {
			return err
		}
	case "fix":
		if err := Fix(opts); err != nil {
			return err
		}
	case "redo":
		if err := Redo(opts); err != nil {
			return err
		}
	case "reset":
		if err := Reset(opts); err != nil {
			return err
		}
	case "status":
		if err := Status(opts); err != nil {
			return err
		}
	case "version":
		if err := Version(opts); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%q: no such command", command)
	}
	return nil
}
