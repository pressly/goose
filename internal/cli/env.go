package cli

import (
	"os"
	"strings"
)

var (
	GOOSE_DBSTRING = envOr("GOOSE_DBSTRING", "")
	GOOSE_DIR      = envOr("GOOSE_DIR", DefaultDir)

	// https://no-color.org/
	NOCOLOR = envOr("NO_COLOR", "false")
)

var (
	DefaultDir = "./migrations"
)

// An EnvVar is an environment variable Name=Value.
type EnvVar struct {
	Value string
	Name  string
}

func List() []EnvVar {
	all := os.Environ()
	envs := []EnvVar{
		{Value: GOOSE_DBSTRING, Name: "GOOSE_DBSTRING"},
		{Value: GOOSE_DIR, Name: "GOOSE_DIR"},
		{Value: NOCOLOR, Name: "NO_COLOR"},
	}
	for _, e := range all {
		if strings.HasPrefix(e, "GOOSE_") {
			name, value, ok := strings.Cut(e, "=")
			if ok {
				envs = append(envs, EnvVar{Name: name, Value: value})
			}
		}
	}
	return envs
}

// envOr returns os.Getenv(key) if set, or else default.
func envOr(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		val = def
	}
	return val
}
