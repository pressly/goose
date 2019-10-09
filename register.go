package goose

import (
	"database/sql"
	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrNoRegisteredMigrations = errors.New("goose: no registered migrations to run.")
)

// Register contains a map of Go migrations that have been registered via goose.AddMigration()
type Register struct {
	goMigrations map[int64]*Migration
}

// globalRegister of Go migrations.
var globalRegister = &Register{
	goMigrations: map[int64]*Migration{},
}

// Registered returns registered go migrations
func Registered() *Register {
	return globalRegister
}

func (r *Register) Run(command string, db *sql.DB) error {
	switch command {
	case "up":
		return r.Up(db)

	case "down":
		return r.Down(db)

	default:
		return fmt.Errorf("%q: no such command", command)
	}
}

// Up runs an up migration of all registered migrations.
func (r *Register) Up(db *sql.DB) error {
	// Ensure there are registered migrations
	if len(globalRegister.goMigrations) == 0 {
		return ErrNoRegisteredMigrations
	}

	// Create migrations from our registered go migrations.
	var migrations Migrations
	for _, migration := range globalRegister.goMigrations {
		migrations = append(migrations, migration)
	}

	for {
		current, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == ErrNoNextVersion {
				log.Printf("goose: no migrations to run. current version: %d\n", current)
				return nil
			}
			return err
		}

		if err = next.Up(db); err != nil {
			return err
		}
	}
}

// Down rolls back a single migration from the current version.
func (r *Register) Down(db *sql.DB) error {
	currentVersion, err := GetDBVersion(db)
	if err != nil {
		return err
	}

	// Go migrations registered via goose.AddMigration().
	var migrations Migrations
	for _, migration := range globalRegister.goMigrations {
		v, err := NumericComponent(migration.Source)
		if err != nil {
			return err
		}
		if versionFilter(v, minVersion, maxVersion) {
			migrations = append(migrations, migration)
		}
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return fmt.Errorf("no migration %v", currentVersion)
	}

	return current.Down(db)
}
