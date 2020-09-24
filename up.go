package goose

// UpTo migrates up to a specific version.
func UpTo(cfg *config, version int64) error {
	migrations, err := CollectMigrations(cfg, minVersion, version)
	if err != nil {
		return err
	}

	for {
		current, err := GetDBVersion(cfg.db)
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

		if err = next.Up(cfg); err != nil {
			return err
		}
	}
}

// Up applies all available migrations.
func Up(cfg *config) error {
	return UpTo(cfg, maxVersion)
}

// UpByOne migrates up by a single version.
func UpByOne(cfg *config) error {
	migrations, err := CollectMigrations(cfg, minVersion, maxVersion)
	if err != nil {
		return err
	}

	currentVersion, err := GetDBVersion(cfg.db)
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

	if err = next.Up(cfg); err != nil {
		return err
	}

	return nil
}
