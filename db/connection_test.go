package db

import (
	"github.com/geniusmonkey/gander/creds"
	"github.com/geniusmonkey/gander/env"
	"testing"
)

func Test_buildMysql(t *testing.T) {
	tests := []struct {
		name string
		env  env.Environment
		cred creds.Credentials
		want string
	}{
		{
			name: "dsn",
			env:  env.Environment{Protocol: "tcp", Port: 3306, Host: "127.0.0.1", Schema: "db", Paramas: map[string]string{}},
			cred: creds.Credentials{},
			want: "tcp(127.0.0.1:3306)/db?",
		},
		{
			name: "dsn_params",
			env: env.Environment{Protocol: "tcp", Port: 3306, Host: "127.0.0.1", Schema: "db",
				Paramas: map[string]string{"parseTime": "1"}},
			cred: creds.Credentials{},
			want: "tcp(127.0.0.1:3306)/db?parseTime=1&",
		},
		{
			name: "dsn_creds",
			env: env.Environment{Protocol: "tcp", Port: 3306, Host: "127.0.0.1", Schema: "db",
				Paramas: map[string]string{"parseTime": "1"}},
			cred: creds.Credentials{Username: "app", Password: "secret"},
			want: "app:secret@tcp(127.0.0.1:3306)/db?parseTime=1&",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildMysql(tt.env, tt.cred); got != tt.want {
				t.Errorf("buildMysql() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildPostgres(t *testing.T) {
	tests := []struct {
		name string
		env  env.Environment
		cred creds.Credentials
		want string
	}{
		{
			name: "dsn",
			env:  env.Environment{Port: 3306, Host: "127.0.0.1", Schema: "db", Paramas: map[string]string{}},
			cred: creds.Credentials{},
			want: "postgres://127.0.0.1:3306/db?",
		},
		{
			name: "dsn_params",
			env: env.Environment{Port: 3306, Host: "127.0.0.1", Schema: "db",
				Paramas: map[string]string{"sslmode": "required"}},
			cred: creds.Credentials{},
			want: "postgres://127.0.0.1:3306/db?sslmode=required&",
		},
		{
			name: "dsn_creds",
			env: env.Environment{Port: 3306, Host: "127.0.0.1", Schema: "db",
				Paramas: map[string]string{"sslmode": "required"}},
			cred: creds.Credentials{Username: "app", Password: "secret"},
			want: "postgres://app:secret@127.0.0.1:3306/db?sslmode=required&",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildPostgres(tt.env, tt.cred); got != tt.want {
				t.Errorf("buildPostgres() = %v, want %v", got, tt.want)
			}
		})
	}
}
