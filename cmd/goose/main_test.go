package main

import (
	"strings"
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

// TestUsageCommandsValidate ensures the validate command description makes clear
// that it checks the structure of migration files and does not validate the SQL
// queries themselves.
func TestUsageCommandsValidate(t *testing.T) {
	for _, line := range strings.Split(usageCommands, "\n") {
		fields := strings.Fields(line)
		if len(fields) > 1 && fields[0] == "validate" {
			if !strings.Contains(line, "not SQL queries") {
				t.Errorf("validate description must state it does not validate SQL queries, got: %q", line)
			}
			return
		}
	}
	t.Error("validate command not found in usageCommands")
}
