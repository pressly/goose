package goose

import (
	"testing"

	"github.com/pressly/goose/v4/internal/check"
)

func TestNumericComponent(t *testing.T) {
	t.Parallel()

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		tt := []struct {
			in   string
			want int64
		}{
			{in: "001_add_updated_at_to_users_table.sql", want: 1},
			{in: "001_add_updated_at_to_users_table.go", want: 1},
			{in: "20230329084915_dir.go", want: 20230329084915},
			{in: "1_.sql", want: 1},
			{in: "010_.sql", want: 10},
		}
		for _, test := range tt {
			got, err := NumericComponent(test.in)
			check.NoError(t, err)
			if got != test.want {
				t.Errorf("unexpected numeric component for input(%q): got %d, want %d", test.in, got, test.want)
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		tt := []struct {
			in string
		}{
			{in: ""},
			{in: "_"},
			{in: "_.sql"},
			{in: "001add_updated_at_to_users_table.sql"},
			{in: "001_add_updated_at_to_users_table"},
			{in: "20230329084915_dir"},
		}
		for _, test := range tt {
			_, err := NumericComponent(test.in)
			if err == nil {
				t.Errorf("expected error for input(%q)", test.in)
			}
		}
	})

}
