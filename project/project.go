package project

import (
	"errors"
	"fmt"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
	"os"
	"path"
)

var fs afero.Fs
var ErrProjectExists = errors.New("existing project")
var IsNotExists = errors.New("not a gander project dir")

const dirName = ".gander"

func init() {
	fs = afero.NewOsFs()
}

type Project struct {
	Driver     string `yaml:"driver"`
	Name       string `yaml:"name"`
	DefaultEnv string `yaml:"default"`
	RootDir    string `yaml:"-"`
	Migrations string `yaml:"migrations"`
}

func (p Project) MigrationDir() string {
	return path.Join(p.RootDir, p.Migrations)
}

func Init(dir string, project Project) error {
	ganderDir := path.Join(dir, dirName)
	_, err := fs.Stat(ganderDir)
	if err == nil {
		return ErrProjectExists
	}

	err = fs.Mkdir(ganderDir, os.ModeDir|os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %v, %w", ganderDir, err)
	}

	stat, err := fs.Stat(dir)
	if err != nil {
		return err
	}
	project.Name = stat.Name()

	f, err := fs.Create(path.Join(ganderDir, "project.yaml"))
	if err != nil {
		return err
	}

	return yaml.NewEncoder(f).Encode(project)
}

func Get(dir string) (*Project, error) {
	ganderDir := path.Join(dir, dirName, "project.yaml")
	file, err := fs.Open(ganderDir)
	if err != nil {
		return nil, IsNotExists
	}

	var p = new(Project)
	err = yaml.NewDecoder(file).Decode(p)
	p.RootDir = dir
	return p, err
}
