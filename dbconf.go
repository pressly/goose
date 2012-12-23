package main

import (
	"fmt"
	"github.com/kylelemons/go-gypsy/yaml"
)

type DBConf struct {
	Name    string
	Driver  string
	OpenStr string
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
