package creds

import (
	"github.com/geniusmonkey/gander/env"
	"github.com/geniusmonkey/gander/project"
	"github.com/geniusmonkey/gander/testing/helper"
	"github.com/spf13/afero"
	"os"
	"path"
	"testing"
)

func TestSave(t *testing.T) {
	defaultPro := project.Project{Name: "test"}
	defaultEnv := env.Environment{Name: "local"}
	defaultCreds := Credentials{Username: "dbuser", Password: "secret"}

	cfgDir, _ := os.UserConfigDir()

	osFs := afero.NewOsFs()
	fs = afero.NewMemMapFs()

	tests := map[string]struct {
		project project.Project
		env     env.Environment
		creds   []Credentials
		golden  string
		err     error
	}{
		"singleEnv": {project: defaultPro, env: defaultEnv, golden: "test-single.golden", creds: []Credentials{defaultCreds}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			for _, cred := range tt.creds {
				err := Save(tt.project, tt.env, cred)
				if err != nil {
					t.Error(err)
				}
			}

			golden := helper.MustReadAll(t, osFs, path.Join("../testdata", tt.golden))
			act := helper.MustReadAll(t, fs, path.Join(cfgDir, dirName, tt.project.Name))
			helper.Equals(t, golden, act)
		})
	}
}
