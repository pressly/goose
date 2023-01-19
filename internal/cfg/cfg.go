package cfg

import "os"

var (
	GOOSEDRIVER       = envOr("GOOSE_DRIVER", "")
	GOOSEDBSTRING     = envOr("GOOSE_DBSTRING", "")
	GOOSEMIGRATIONDIR = envOr("GOOSE_MIGRATION_DIR", DefaultMigrationDir)
	// https://no-color.org/
	GOOSENOCOLOR = envOr("NO_COLOR", "false")
)

var (
	DefaultMigrationDir = "."
)

// An EnvVar is an environment variable Name=Value.
type EnvVar struct {
	Name  string
	Value string
}

func List() []EnvVar {
	return []EnvVar{
		{Name: "GOOSE_DRIVER", Value: GOOSEDRIVER},
		{Name: "GOOSE_DBSTRING", Value: GOOSEDBSTRING},
		{Name: "GOOSE_MIGRATION_DIR", Value: GOOSEMIGRATIONDIR},
		{Name: "NO_COLOR", Value: GOOSENOCOLOR},
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
