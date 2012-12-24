package main

import (
	"flag"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"path"
)

// global options. available to any subcommands.
var dbFolder = flag.String("db", "db", "folder containing db info")
var dbConfName = flag.String("config", "development", "which DB configuration to use")

type DBConf struct {
	MigrationsDir string
	Name          string
	Driver        string
	OpenStr       string
}

// extract configuration details from the given file
func MakeDBConf() (*DBConf, error) {

	cfgFile := path.Join(*dbFolder, "dbconf.yml")

	f, err := yaml.ReadFile(cfgFile)
	if err != nil {
		return nil, err
	}

	drv, derr := f.Get(fmt.Sprintf("%s.driver", *dbConfName))
	if derr != nil {
		return nil, derr
	}

	open, oerr := f.Get(fmt.Sprintf("%s.open", *dbConfName))
	if oerr != nil {
		return nil, oerr
	}

	return &DBConf{
		MigrationsDir: path.Join(*dbFolder, "migrations"),
		Name:          *dbConfName,
		Driver:        drv,
		OpenStr:       open,
	}, nil
}
