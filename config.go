package gander

type Config struct {
	Environments map[string]Environment `toml:"env"`
}

type Environment struct {
	Driver       string
	Dsn          string
	MigrationsDir string
}
