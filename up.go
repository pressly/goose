package goose

// UpTo migrates up to a specific version.
func UpTo(opts *options, version int64) error {
	migrations, err := CollectMigrations(opts, minVersion, version)
	if err != nil {
		return err
	}

	for {
		current, err := GetDBVersion(opts.db)
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

		if err = next.Up(opts.db); err != nil {
			return err
		}
	}
}

// Up applies all available migrations.
func Up(opts *options) error {
	return UpTo(opts, maxVersion)
}

// UpByOne migrates up by a single version.
func UpByOne(opts *options) error {
	migrations, err := CollectMigrations(opts, minVersion, maxVersion)
	if err != nil {
		return err
	}

	currentVersion, err := GetDBVersion(opts.db)
	if err != nil {
		return err
	}

	next, err := migrations.Next(currentVersion)
	if err != nil {
		if err == ErrNoNextVersion {
			log.Printf("goose: no migrations to run. current version: %d\n", currentVersion)
		}
		return err
	}

	if err = next.Up(opts.db); err != nil {
		return err
	}

	return nil
}
