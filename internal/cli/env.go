package cli

import (
	"os"
	"strings"
)

var (
	// https://no-color.org/
	GOOSE_NOCOLOR = envOr("NO_COLOR", "false")
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
		{Value: GOOSE_NOCOLOR, Name: "NO_COLOR"},
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
