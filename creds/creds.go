package creds

import (
	"errors"
	"fmt"
	"github.com/apex/log"
	"github.com/geniusmonkey/gander/env"
	"github.com/geniusmonkey/gander/project"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"os"
	"path"
)

var (
	fs         = afero.NewOsFs()
	ganderDir  string
	IsNotExist = errors.New("credentials do not exist")
)

const dirName = "gander"

type projectCreds map[string]Credentials

type Credentials struct {
	Username string
	Password string
}

func init() {
	usrDir, err := os.UserConfigDir()
	if err != nil {
		panic(fmt.Errorf("failed to locate user dir, %v", err))
	}

	ganderDir = path.Join(usrDir, dirName)
}

func Save(proj project.Project, environment env.Environment, credentials Credentials) error {
	pc, err := loadProjCreds(proj)
	if err != nil {
		return err
	}

	pc[environment.Name] = credentials

	credPath := path.Join(ganderDir, proj.Name)

	var file afero.File
	if exists, err := afero.Exists(fs, credPath); err != nil {
		return err
	} else if exists {
		file, err = fs.OpenFile(credPath, os.O_TRUNC|os.O_WRONLY, 0)
	} else {
		if err := fs.MkdirAll(ganderDir, os.ModeDir|os.ModePerm); err != nil {
			return fmt.Errorf("unable to create config directory, %w", err)
		}
		file, err = fs.Create(credPath)
		if err != nil {
			return err
		}
	}

	yaml, err := yaml.Marshal(pc)
	if err != nil {
		return err
	}

	if _, err = file.Write(yaml); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return nil
}

func Get(proj project.Project, environment env.Environment) (Credentials, error) {
	pc, err := loadProjCreds(proj)
	if err != nil {
		return Credentials{}, err
	}

	if c, ok := pc[environment.Name]; ok {
		return c, nil
	} else {
		return Credentials{}, IsNotExist
	}
}

func loadProjCreds(proj project.Project) (projectCreds, error) {
	pc := make(map[string]Credentials)

	credPath := path.Join(ganderDir, proj.Name)
	if exists, err := afero.Exists(fs, credPath); err != nil {
		return pc, err
	} else if exists {
		log.Debugf("using credentials file found at $s", credPath)
		file, err := fs.Open(credPath)
		if err != nil {
			return pc, err
		}
		err = yaml.NewDecoder(file).Decode(&pc)
		return pc, err
	} else {
		log.Debugf("credential file %s does not exist returning empty config", credPath)
		return pc, nil
	}
}
