package goose

import (
	"testing"
)

func TestCamelSnake(t *testing.T) {
	tt := []struct {
		in    string
		camel string
		snake string
	}{
		{in: "Add updated_at to users table", camel: "addUpdatedAtToUsersTable", snake: "add_updated_at_to_users_table"},
		{in: "$()&^%(_--crazy__--input$)", camel: "crazyInput", snake: "crazy_input"},
	}

	for _, test := range tt {
		if got := lowerCamelCase(test.in); got != test.camel {
			t.Errorf("unexpected lowerCamelCase for input(%q), got %q, want %q", test.in, got, test.camel)
		}
		if got := snakeCase(test.in); got != test.snake {
			t.Errorf("unexpected snake_case for input(%q), got %q, want %q", test.in, got, test.snake)
		}
	}
}