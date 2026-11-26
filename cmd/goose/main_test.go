package main

import (
	"testing"
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

// TestMergeArgsIssue937Regression covers argument precedence and migration
// commands with their own positional operands (issue #937).
func TestMergeArgsIssue937Regression(t *testing.T) {
	env := &envConfig{driver: "postgres", dbstring: "postgresql://env/db"}
	tests := []struct {
		name string
		cfg  *envConfig
		args []string
		want []string
	}{
		{
			name: "command only uses env values",
			cfg:  env,
			args: []string{"status"},
			want: []string{"postgres", "postgresql://env/db", "status"},
		},
		{
			name: "command argument is not mistaken for DBSTRING",
			cfg:  env,
			args: []string{"up-to", "42"},
			want: []string{"postgres", "postgresql://env/db", "up-to", "42"},
		},
		{
			name: "explicit driver and DBSTRING override env",
			cfg:  env,
			args: []string{"sqlite3", "./cli.db", "status"},
			want: []string{"sqlite3", "./cli.db", "status"},
		},
		{
			name: "explicit connection values with command operand",
			cfg:  env,
			args: []string{"sqlite3", "./cli.db", "up-to", "42"},
			want: []string{"sqlite3", "./cli.db", "up-to", "42"},
		},
		{
			name: "CLI driver overrides env driver",
			cfg:  env,
			args: []string{"sqlite3", "status"},
			want: []string{"sqlite3", "postgresql://env/db", "status"},
		},
		{
			name: "CLI driver with command operand",
			cfg:  env,
			args: []string{"postgres", "up-to", "42"},
			want: []string{"postgres", "postgresql://env/db", "up-to", "42"},
		},
		{
			name: "CLI DBSTRING overrides env DBSTRING",
			cfg:  env,
			args: []string{"postgresql://cli/db", "status"},
			want: []string{"postgres", "postgresql://cli/db", "status"},
		},
		{
			name: "CLI DBSTRING with command operand",
			cfg:  env,
			args: []string{"postgresql://cli/db", "down-to", "42"},
			want: []string{"postgres", "postgresql://cli/db", "down-to", "42"},
		},
		{
			name: "explicit CLI without env",
			cfg:  &envConfig{},
			args: []string{"postgres", "postgresql://cli/db", "status"},
			want: []string{"postgres", "postgresql://cli/db", "status"},
		},
		{
			name: "command without env stays incomplete",
			cfg:  &envConfig{},
			args: []string{"status"},
			want: []string{"status"},
		},
		{
			name: "driver-only env and CLI DBSTRING",
			cfg:  &envConfig{driver: "postgres"},
			args: []string{"postgresql://cli/db", "status"},
			want: []string{"postgres", "postgresql://cli/db", "status"},
		},
		{
			name: "DBSTRING-only env and CLI driver",
			cfg:  &envConfig{dbstring: "postgresql://env/db"},
			args: []string{"postgres", "status"},
			want: []string{"postgres", "postgresql://env/db", "status"},
		},
		{
			name: "empty args preserved",
			cfg:  env,
			args: []string{},
			want: []string{},
		},
		{
			name: "unknown command forwarded when env is configured",
			cfg:  env,
			args: []string{"not-a-goose-command"},
			want: []string{"postgres", "postgresql://env/db", "not-a-goose-command"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeArgs(tt.cfg, tt.args)
			if len(got) != len(tt.want) {
				t.Fatalf("mergeArgs() = %q, want %q", got, tt.want)
			}
			for i, value := range tt.want {
				if got[i] != value {
					t.Fatalf("mergeArgs() = %q, want %q", got, tt.want)
				}
			}
		})
	}
}
