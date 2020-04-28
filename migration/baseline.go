package migration

import (
	"database/sql"
)

func Baseline(db *sql.DB, dir string, ver int64) error {
	migrations, err := CollectMigrations(dir, minVersion, ver)
	if err != nil {
		return err
	}

	for {
		current, err := GetDBVersion(db)
		if err != nil {
			return err
		}

		next, err := migrations.Next(current)
		if err != nil {
			if err == ErrNoNextVersion {
				log.Infof("goose: no migrations to run. current version: %d\n", current)
				return nil
			}
			return err
		}

		if err = next.Baseline(db); err != nil {
			return err
		}
	}
}