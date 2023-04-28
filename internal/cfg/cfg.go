package cfg

import (
	"os"
	"strings"
)

var (
	GOOSEDRIVER       = envOr("GOOSE_DRIVER", "")
	GOOSEDBSTRING     = envOr("GOOSE_DBSTRING", "")
	GOOSEMIGRATIONDIR = envOr("GOOSE_MIGRATION_DIR", DefaultMigrationDir)
	// https://no-color.org/
	GOOSENOCOLOR = envOr("NO_COLOR", "false")

	GOOSE_CLICKHOUSE_PARAMS = envOr("GOOSE_CLICKHOUSE_PARAMS", "")
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
		{Name: "GOOSE_CLICKHOUSE_PARAMS", Value: GOOSE_CLICKHOUSE_PARAMS},
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

func SplitKeyValuesIntoMap(input string) map[string]string {
	cutset := strings.Split(input, ",")

	options := make(map[string]string)
	for _, item := range cutset {
		keyAndValues := strings.Split(item, "=")
		if len(keyAndValues) != 2 {
			continue
		}
		options[keyAndValues[0]] = keyAndValues[1]
	}
	return options
}
