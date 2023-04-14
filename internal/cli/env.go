package cli

import (
	"os"
)

const (
	DefaultTableName = "goose_db_version"
)

const (
	EnvGooseDBString string = "GOOSE_DBSTRING"
	EnvGooseDir      string = "GOOSE_DIR"
	EnvGooseTable    string = "GOOSE_TABLE"
	EnvNoColor       string = "NO_COLOR"
)

var (
	GOOSE_DBSTRING = envOr(EnvGooseDBString, "")
	GOOSE_TABLE    = envOr(EnvGooseTable, DefaultTableName)
	GOOSE_DIR      = envOr(EnvGooseDir, "")

	// https://no-color.org/
	NOCOLOR = envOr(EnvNoColor, "false")

	envLookup = map[string]string{
		EnvGooseDBString: "Database connection string, lower priority than --dbstring",
		EnvGooseDir:      "Directory with migration files, lower priority than --dir",
		EnvGooseTable:    `Database table name, lower priority than --table (default "goose_db_version")`,
		EnvNoColor:       "Disable color output, lower priority than --no-color",
	}
)

// An EnvVar is an environment variable Name=Value.
type EnvVar struct {
	Value string
	Name  string
}

func List() []EnvVar {
	return []EnvVar{
		{Value: GOOSE_DBSTRING, Name: EnvGooseDBString},
		{Value: GOOSE_DIR, Name: EnvGooseDir},
		{Value: GOOSE_TABLE, Name: EnvGooseTable},
		{Value: NOCOLOR, Name: EnvNoColor},
	}
}

// envOr returns os.Getenv(key) if set, or else default.
func envOr(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		val = def
	}
	return val
}
