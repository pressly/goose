package goose

import (
	"database/sql"
	"fmt"
	"strconv"
	"sync"
)

const VERSION = "v2.7.0-rc3"

var (
	duplicateCheckOnce sync.Once
	minVersion         = int64(0)
	maxVersion         = int64((1 << 63) - 1)
	timestampFormat    = "20060102150405"
)

// SetVerbose set the goose verbosity mode
func SetVerbose(v bool) { def.SetVerbose(v) }

// SetVerbose set the goose verbosity mode
func (in *Instance) SetVerbose(v bool) {
	in.verbose = v
}

// Run runs a goose command.
func Run(command string, db *sql.DB, dir string, args ...string) error {
	return def.Run(command, db, dir, args...)
}

// Run runs a goose command.
func (in *Instance) Run(command string, db *sql.DB, dir string, args ...string) error {
	switch command {
	case "up":
		if err := in.Up(db, dir); err != nil {
			return err
		}
	case "up-by-one":
		if err := in.UpByOne(db, dir); err != nil {
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
		if err := in.UpTo(db, dir, version); err != nil {
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
		if err := in.Create(db, dir, args[0], migrationType); err != nil {
			return err
		}
	case "down":
		if err := in.Down(db, dir); err != nil {
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
		if err := in.DownTo(db, dir, version); err != nil {
			return err
		}
	case "fix":
		if err := in.Fix(dir); err != nil {
			return err
		}
	case "redo":
		if err := in.Redo(db, dir); err != nil {
			return err
		}
	case "reset":
		if err := in.Reset(db, dir); err != nil {
			return err
		}
	case "status":
		if err := in.Status(db, dir); err != nil {
			return err
		}
	case "version":
		if err := in.Version(db, dir); err != nil {
			return err
		}
	default:
		return fmt.Errorf("%q: no such command", command)
	}
	return nil
}
