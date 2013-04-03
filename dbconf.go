package main

import (
	"flag"
	"fmt"
	"github.com/lib/pq"
	"github.com/kylelemons/go-gypsy/yaml"
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

	drv, derr := f.Get(fmt.Sprintf("%s.driver", env))
	if derr != nil {
		return nil, derr
	}

	open, oerr := f.Get(fmt.Sprintf("%s.open", env))
	if oerr != nil {
		return nil, oerr
	}
	open = os.ExpandEnv(open)

	// Automatically parse postgres urls
	if drv == "postgres" {
		parsed_open, parse_err := pq.ParseURL(open)
		// Assumption: If we can parse the URL, we should
		if parse_err == nil && parsed_open != "" {
			open = parsed_open
		}
	}

	return &DBConf{
		MigrationsDir: filepath.Join(p, "migrations"),
		Env:           env,
		Driver:        drv,
		OpenStr:       open,
	}, nil
}
