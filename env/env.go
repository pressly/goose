package env

import (
	"errors"
	"fmt"
	"github.com/geniusmonkey/gander/project"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"path"
)

var fs afero.Fs

func init() {
	fs = afero.NewOsFs()
}

type environments map[string]Environment

type Environment struct {
	Host     string
	Port     int32
	Protocol string
	Schema   string
	Paramas  map[string]string
	Name     string `yaml:"-"`
}

func Get(prj *project.Project, name string) (Environment, error) {
	envs, err := loadEnvs(prj.RootDir)
	if err != nil {
		return Environment{}, err
	}

	if e, ok := envs[name]; ok {
		e.Name = name
		return e, nil
	} else {
		return Environment{}, errors.New("unable to find env with name " + name)
	}
}

func Add(project project.Project, name string, env Environment) error {
	var envs, err = loadEnvs(project.RootDir)
	if err != nil {
		return err
	}

	if _, ok := envs[name]; ok {
		return errors.New("duplicated environment name")
	}

	envs[name] = env
	return saveEnvs(project.RootDir, envs)
}

func Remove(project project.Project, name string) error {
	return nil
}

func saveEnvs(directory string, envs environments) error {
	envFile := path.Join(directory, ".gander", "envs.yaml")
	var file afero.File
	var err error

	_, exists := fs.Stat(envFile)
	if exists != nil {
		file, err = fs.Create(envFile)
	} else {
		file, err = fs.Open(envFile)
	}

	if err != nil {
		return fmt.Errorf("failed to create/open envs.yaml file %w", err)
	}

	err = yaml.NewEncoder(file).Encode(envs)
	if err != nil {
		return fmt.Errorf("failed to read envrionment file, %w", err)
	}
	return nil
}

func loadEnvs(directory string) (environments, error) {
	envs := make(map[string]Environment)
	file, err := fs.Open(path.Join(directory, ".gander", "envs.yaml"))
	if err != nil {
		return make(map[string]Environment), nil
	}

	err = yaml.NewDecoder(file).Decode(&envs)
	if err != nil {
		return envs, fmt.Errorf("failed to read envrionment file, %w", err)
	}
	return envs, nil
}
