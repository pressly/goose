package main

import (
	"strconv"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestFirstNonEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "no values",
			input:    []string{},
			expected: "",
		},
		{
			name:     "all empty values",
			input:    []string{"", "", ""},
			expected: "",
		},
		{
			name:     "single non-empty value at start",
			input:    []string{"value", "", ""},
			expected: "value",
		},
		{
			name:     "single non-empty value in middle",
			input:    []string{"", "value", ""},
			expected: "value",
		},
		{
			name:     "single non-empty value at end",
			input:    []string{"", "", "value"},
			expected: "value",
		},
		{
			name:     "multiple non-empty values",
			input:    []string{"first", "second", "third"},
			expected: "first",
		},
		{
			name:     "mixed empty and non-empty values",
			input:    []string{"", "value1", "", "value2"},
			expected: "value1",
		},
		{
			name:     "only one value, empty",
			input:    []string{""},
			expected: "",
		},
		{
			name:     "only one value, non-empty",
			input:    []string{"value"},
			expected: "value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firstNonEmpty(tt.input...)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// TestInsertOmittedDBString covers the merged positional arguments when the
// DBSTRING is omitted (issue #1169). postgres and redshift are backed by pgx,
// which reads the standard libpq environment variables, so their DRIVER
// COMMAND form is expanded to DRIVER "" COMMAND. All other drivers must keep
// supplying a DBSTRING and fail the len(args) < 3 check in main.
func TestInsertOmittedDBString(t *testing.T) {
	tests := []struct {
		name string
		cfg  *envConfig
		args []string
		want []string
	}{
		{
			name: "postgres command",
			cfg:  &envConfig{},
			args: []string{"postgres", "up"},
			want: []string{"postgres", "", "up"},
		},
		{
			name: "redshift command",
			cfg:  &envConfig{},
			args: []string{"redshift", "status"},
			want: []string{"redshift", "", "status"},
		},
		{
			name: "env driver",
			cfg:  &envConfig{driver: "postgres"},
			args: []string{"status"},
			want: []string{"postgres", "", "status"},
		},
		{
			name: "env driver redshift",
			cfg:  &envConfig{driver: "redshift"},
			args: []string{"status"},
			want: []string{"redshift", "", "status"},
		},
		{
			name: "env driver and dbstring win",
			cfg:  &envConfig{driver: "postgres", dbstring: "host=env"},
			args: []string{"status"},
			want: []string{"postgres", "host=env", "status"},
		},
		{
			name: "full positional triple unchanged",
			cfg:  &envConfig{},
			args: []string{"postgres", "host=cli", "up"},
			want: []string{"postgres", "host=cli", "up"},
		},
		{
			name: "postgres three tokens unchanged",
			cfg:  &envConfig{},
			args: []string{"postgres", "up-to", "10"},
			want: []string{"postgres", "up-to", "10"},
		},
		{
			name: "sqlite3 still requires dbstring",
			cfg:  &envConfig{},
			args: []string{"sqlite3", "up"},
			want: []string{"sqlite3", "up"},
		},
		{
			name: "mysql still requires dbstring",
			cfg:  &envConfig{driver: "mysql"},
			args: []string{"up"},
			want: []string{"mysql", "up"},
		},
		{
			name: "single token unchanged",
			cfg:  &envConfig{},
			args: []string{"postgres"},
			want: []string{"postgres"},
		},
		{
			name: "no args unchanged",
			cfg:  &envConfig{},
			args: []string{},
			want: []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := insertOmittedDBString(mergeArgs(tt.cfg, tt.args))
			if len(got) != len(tt.want) {
				t.Fatalf("insertOmittedDBString(mergeArgs()) = %q, want %q", got, tt.want)
			}
			for i, value := range tt.want {
				if got[i] != value {
					t.Fatalf("insertOmittedDBString(mergeArgs()) = %q, want %q", got, tt.want)
				}
			}
		})
	}
}

// TestPgxEmptyConnStringReadsEnv verifies the pgx behavior this change relies
// on: parsing an empty connection string picks up the standard libpq
// environment variables (jackc/pgx/v5 v5.12.0).
func TestPgxEmptyConnStringReadsEnv(t *testing.T) {
	t.Setenv("PGHOST", "pg.env.local")
	t.Setenv("PGPORT", "5544")
	t.Setenv("PGUSER", "envuser")
	t.Setenv("PGPASSWORD", "envpass")
	t.Setenv("PGDATABASE", "envdb")

	config, err := pgx.ParseConfig("")
	if err != nil {
		t.Fatalf("pgx.ParseConfig(\"\") = %v", err)
	}
	for name, tc := range map[string]struct {
		got  string
		want string
	}{
		"PGHOST":     {config.Host, "pg.env.local"},
		"PGPORT":     {strconv.Itoa(int(config.Port)), "5544"},
		"PGUSER":     {config.User, "envuser"},
		"PGPASSWORD": {config.Password, "envpass"},
		"PGDATABASE": {config.Database, "envdb"},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", name, tc.got, tc.want)
		}
	}
}
