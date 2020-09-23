package goose

// Redo rolls back the most recently applied migration, then runs it again.
func Redo(opts *options) error {
	currentVersion, err := GetDBVersion(opts.db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(opts, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return err
	}

	if err := current.Down(opts.db); err != nil {
		return err
	}

	if err := current.Up(opts.db); err != nil {
		return err
	}

	return nil
}
