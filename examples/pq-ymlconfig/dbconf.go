package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/lib/pq"
)

type DBConf struct {
	DBString      string
	Driver        string
	MigrationsDir string
	Env           string
	PgSchema      string
}

// extract configuration details from the given file
// inputs
//	path: folder containing db info
//  env: which DB environment to use
//  pgschema: which postgres-schema to migrate (default = none)
func NewDBConf(path, env string, pgschema string) (*DBConf, error) {
	cfgFile := filepath.Join(path, "dbconf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	// database driver
	drv, err := f.Get(fmt.Sprintf("%s.driver", env))
	if err != nil {
		return nil, err
	}
	drv = os.ExpandEnv(drv)

	// database string
	open, err := f.Get(fmt.Sprintf("%s.open", env))
	if err != nil {
		return nil, err
	}
	open = os.ExpandEnv(open)

	// Automatically parse postgres urls
	// Assumption: If we can parse the URL, we should
	if parsedURL, err := pq.ParseURL(open); err == nil && parsedURL != "" {
		open = parsedURL
	}

	return &DBConf{
		DBString:      open,
		Driver:        drv,
		MigrationsDir: path,
		Env:           env,
		PgSchema:      pgschema,
	}, nil
}
