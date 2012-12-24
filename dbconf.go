package main

import (
	"flag"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"path"
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

// extract configuration details from the given file
func MakeDBConf() (*DBConf, error) {

	cfgFile := path.Join(*dbPath, "dbconf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	drv, derr := f.Get(fmt.Sprintf("%s.driver", *dbEnv))
	if derr != nil {
		return nil, derr
	}

	open, oerr := f.Get(fmt.Sprintf("%s.open", *dbEnv))
	if oerr != nil {
		return nil, oerr
	}

	return &DBConf{
		MigrationsDir: path.Join(*dbPath, "migrations"),
		Env:           *dbEnv,
		Driver:        drv,
		OpenStr:       open,
	}, nil
}
