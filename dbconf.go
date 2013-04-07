package main

import (
	"flag"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"github.com/lib/pq"
	"os"
	"path/filepath"
)

// global options. available to any subcommands.
var dbPath = flag.String("path", "db", "folder containing db info")
var dbEnv = flag.String("env", "development", "which DB environment to use")

type DBConf struct {
	MigrationsDir string
	Env           string
	Driver        string
	OpenStr       string
}

// default helper - makes a DBConf from the dbPath and dbEnv flags
func MakeDBConf() (*DBConf, error) {
	return makeDBConfDetails(*dbPath, *dbEnv)
}

// extract configuration details from the given file
func makeDBConfDetails(p, env string) (*DBConf, error) {

	cfgFile := filepath.Join(p, "dbconf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	drv, err := f.Get(fmt.Sprintf("%s.driver", env))
	if err != nil {
		return nil, err
	}

	open, err := f.Get(fmt.Sprintf("%s.open", env))
	if err != nil {
		return nil, err
	}
	open = os.ExpandEnv(open)

	// Automatically parse postgres urls
	if drv == "postgres" {

		// Assumption: If we can parse the URL, we should
		if parsedURL, err := pq.ParseURL(open); err == nil && parsedURL != "" {
			open = parsedURL
		}
	}

	return &DBConf{
		MigrationsDir: filepath.Join(p, "migrations"),
		Env:           env,
		Driver:        drv,
		OpenStr:       open,
	}, nil
}
