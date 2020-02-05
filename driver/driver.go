package driver

import (
	"fmt"
	"github.com/geniusmonkey/gander/env"
)

var defaults = map[string]env.Environment{
	"mysql": {
		Host:     "127.0.0.1",
		Port:     3306,
		Protocol: "tcp",
		Schema:   "database",
		Paramas: map[string]string{
			"parseTime": "true",
		},
	},
	"cockroach": {
		Host:   "127.0.0.1",
		Port:   26257,
		Schema: "database",
		Paramas: map[string]string{
			"sslmode":     "require",
			"sslrootcert": "app.crt",
		},
	},
	"redshift": {
		Host:   "127.0.0.1",
		Port:   26257,
		Schema: "database",
		Paramas: map[string]string{
		},
	},
}

func DefaultEnv(name string) (env.Environment, error) {
	if e, ok := defaults[name]; ok {
		return e, nil
	} else {
		return env.Environment{}, fmt.Errorf("default environment %v not found", name)
	}
}
