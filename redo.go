package goose

// Redo rolls back the most recently applied migration, then runs it again.
func Redo(cfg *config) error {
	currentVersion, err := GetDBVersion(cfg.db)
	if err != nil {
		return err
	}

	migrations, err := CollectMigrations(cfg, minVersion, maxVersion)
	if err != nil {
		return err
	}

	current, err := migrations.Current(currentVersion)
	if err != nil {
		return err
	}

	if err := current.Down(cfg); err != nil {
		return err
	}

	if err := current.Up(cfg); err != nil {
		return err
	}

	return nil
}
