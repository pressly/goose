package main

import (
	"flag"
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
	"log"
	"path"
)

type DBConf struct {
	Name    string
	Driver  string
	OpenStr string
}

var dbFolder = flag.String("db", "db", "folder containing db info")
var dbConfName = flag.String("config", "development", "which DB configuration to use")
var targetVersion = flag.Int("target", -1, "which DB version to target (defaults to latest version)")

func main() {
	flag.Parse()

	conf, err := dbConfFromFile(path.Join(*dbFolder, "dbconf.yml"), *dbConfName)
	if err != nil {
		log.Fatal(err)
	}

	runMigrations(conf, path.Join(*dbFolder, "migrations"), *targetVersion)
}

// extract configuration details from the given file
func dbConfFromFile(path, envtype string) (*DBConf, error) {

	f, err := yaml.ReadFile(path)
	if err != nil {
		return nil, err
	}

	drv, derr := f.Get(fmt.Sprintf("%s.driver", envtype))
	if derr != nil {
		return nil, derr
	}

	open, oerr := f.Get(fmt.Sprintf("%s.open", envtype))
	if oerr != nil {
		return nil, oerr
	}

	return &DBConf{
		Name:    envtype,
		Driver:  drv,
		OpenStr: open,
	}, nil
}
