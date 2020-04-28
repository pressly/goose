package creds

import (
	"github.com/geniusmonkey/gander/env"
	"github.com/geniusmonkey/gander/project"
	"github.com/geniusmonkey/gander/testing/helper"
	"github.com/spf13/afero"
	"path"
	"reflect"
	"testing"
)

var (
	defaultPro = project.Project{Name: "test"}
	envLocal   = env.Environment{Name: "local"}
	envDev     = env.Environment{Name: "dev"}
	credsLocal = Credentials{Username: "dbuser", Password: "secret-local"}
	credsDev   = Credentials{Username: "dbuser", Password: "secret-dev"}
)

func TestSave(t *testing.T) {
	type args struct {
		env  env.Environment
		cred Credentials
	}

	osFs := afero.NewOsFs()
	fs = afero.NewMemMapFs()

	tests := map[string]struct {
		project project.Project
		args    []args
		golden  string
	}{
		"Single Env": {project: defaultPro, args: []args{{env: envLocal, cred: credsLocal}}, golden: "creds-single.golden"},
		"Multi Env": {project: defaultPro, golden: "creds-multi.golden", args: []args{
			{env: envLocal, cred: credsLocal},
			{env: envDev, cred: credsDev},
		}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			for _, arg := range tt.args {
				if err := Save(defaultPro, arg.env, arg.cred); err != nil {
					t.Fatal(err)
				}
			}
			golden := helper.MustReadAll(t, osFs, path.Join("../testdata", tt.golden))
			act := helper.MustReadAll(t, fs, path.Join(ganderDir, tt.project.Name))
			helper.Equals(t, string(golden), string(act))
		})
	}
}

func TestGet(t *testing.T) {
	type args struct {
		proj        project.Project
		environment env.Environment
	}
	tests := []struct {
		name    string
		args    args
		want    Credentials
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Get(tt.args.proj, tt.args.environment)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
		})
	}
}
