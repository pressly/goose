package cfg

import "os"

var (
	GOOSEDRIVER       = ""
	GOOSEDBSTRING     = ""
	GOOSEMIGRATIONDIR = DefaultMigrationDir
	// https://no-color.org/
	GOOSENOCOLOR        = "false"
	GOOSEMIGRATIONTABLE = DefaultMigrationTable
)

var (
	DefaultMigrationDir   = "."
	DefaultMigrationTable = "goose_db_version"
)

// Load reads the config values from environment,
// allowing them to be loaded first from file pointed by `-env-file` argument
func Load() {
	GOOSEDRIVER = envOr("GOOSE_DRIVER", GOOSEDRIVER)
	GOOSEDBSTRING = envOr("GOOSE_DBSTRING", GOOSEDBSTRING)
	GOOSEMIGRATIONDIR = envOr("GOOSE_MIGRATION_DIR", GOOSEMIGRATIONDIR)
	// https://no-color.org/
	GOOSENOCOLOR = envOr("NO_COLOR", GOOSENOCOLOR)
	GOOSEMIGRATIONTABLE = envOr("GOOSE_MIGRATION_TABLE", GOOSEMIGRATIONTABLE)
}

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
		{Name: "GOOSE_MIGRATION_TABLE", Value: GOOSEMIGRATIONTABLE},
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
