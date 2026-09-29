package main

import (
	"reflect"
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

func TestMergeArgs(t *testing.T) {
	config := &envConfig{
		driver:   "postgres",
		dbstring: "postgresql://postgres:postgres@localhost:5433/alpha",
	}

	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{
			name:     "empty args",
			args:     []string{},
			expected: []string{},
		},
		{
			name:     "command only uses env driver and dbstring",
			args:     []string{"status"},
			expected: []string{"postgres", "postgresql://postgres:postgres@localhost:5433/alpha", "status"},
		},
		{
			name:     "command with argument uses env driver and dbstring",
			args:     []string{"up-to", "42"},
			expected: []string{"postgres", "postgresql://postgres:postgres@localhost:5433/alpha", "up-to", "42"},
		},
		{
			name:     "cli driver and dbstring override env values",
			args:     []string{"postgres", "postgresql://override", "status"},
			expected: []string{"postgres", "postgresql://override", "status"},
		},
		{
			name:     "cli driver with command uses env dbstring only",
			args:     []string{"postgres", "status"},
			expected: []string{"postgres", "postgresql://postgres:postgres@localhost:5433/alpha", "status"},
		},
		{
			name:     "driver only remains unchanged",
			args:     []string{"postgres"},
			expected: []string{"postgres"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mergeArgs(config, tt.args)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
