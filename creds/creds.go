package creds

import (
	"errors"
	"fmt"
	"github.com/geniusmonkey/gander/env"
	"github.com/geniusmonkey/gander/project"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"os"
	"path"
)

var fs = afero.NewOsFs()
var IsNotExist = errors.New("credentials do not exist")

type projectCreds map[string]Credentials

type Credentials struct {
	Username string
	Password string
}

func Save(proj project.Project, environment env.Environment, credentials Credentials) error {
	pc, err := loadProCreds(proj)
	if err != nil {
		return err
	}

	pc[environment.Name] = credentials

	usrDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	ganderCfg := path.Join(usrDir, "gander")
	credPath := path.Join(ganderCfg, proj.Name)

	var file afero.File
	if ok, err := exists(credPath); err != nil {
		return err
	} else if ok {
		file, err = os.Open(credPath)
	} else {
		if err := fs.MkdirAll(ganderCfg, os.ModeDir|os.ModePerm); err != nil {
			return fmt.Errorf("unable to create config directory, %w", err)
		}
		file, err = os.Create(credPath)
	}

	if err != nil {
		return err
	}

	return yaml.NewEncoder(file).Encode(pc)
}

func Get(proj project.Project, environment env.Environment) (Credentials, error) {
	pc, err := loadProCreds(proj)
	if err != nil {
		return Credentials{}, err
	}

	if c, ok := pc[environment.Name]; ok {
		return c, nil
	} else {
		return Credentials{}, IsNotExist
	}
}

func loadProCreds(proj project.Project) (projectCreds, error) {
	pc := make(map[string]Credentials)
	usrDir, err := os.UserConfigDir()
	if err != nil {
		return pc, err
	}

	credPath := path.Join(usrDir, "gander", proj.Name)
	if ok, err := exists(credPath); err != nil {
		return pc, err
	} else if ok {
		file, err := fs.Open(credPath)
		if err != nil {
			return pc, err
		}
		err = yaml.NewDecoder(file).Decode(&pc)
		return pc, err
	} else {
		return pc, nil
	}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return true, err
}
