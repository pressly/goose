package main

import (
	"slices"
	"testing"
)

func TestMergeArgsPostgresDBString(t *testing.T) {
	for _, test := range []struct {
		name string
		env  envConfig
		args []string
		want []string
	}{
		{"postgres driver flag", envConfig{}, []string{"postgres", "status"}, []string{"postgres", "", "status"}},
		{"redshift driver env", envConfig{driver: "redshift"}, []string{"up"}, []string{"redshift", "", "up"}},
		{"version argument", envConfig{}, []string{"postgres", "up-to", "42"}, []string{"postgres", "", "up-to", "42"}},
		{"absolute down version", envConfig{driver: "postgres"}, []string{"down-to", "0"}, []string{"postgres", "", "down-to", "0"}},
		{"explicit dsn", envConfig{}, []string{"postgres", "host=mysql.example", "status"}, []string{"postgres", "host=mysql.example", "status"}},
		{"env dsn", envConfig{driver: "postgres", dbstring: "host=database"}, []string{"status"}, []string{"postgres", "host=database", "status"}},
		{"sqlite still requires dsn", envConfig{}, []string{"sqlite3", "status"}, []string{"sqlite3", "status"}},
		{"not a command", envConfig{}, []string{"postgres", "upwards"}, []string{"postgres", "upwards"}},
		{"do not rewrite malformed explicit dsn", envConfig{}, []string{"postgres", "down", "status"}, []string{"postgres", "down", "status"}},
		{"distinguish non-numeric command after dsn", envConfig{}, []string{"postgres", "up-to", "status"}, []string{"postgres", "up-to", "status"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := mergeArgs(&test.env, test.args); !slices.Equal(got, test.want) {
				t.Errorf("mergeArgs() = %q, want %q", got, test.want)
			}
		})
	}
}

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
