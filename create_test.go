package goose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/migration"
)

func TestSequential(t *testing.T) {
	t.Cleanup(func() {
		// Reset the global state.
		registeredGoMigrations = make(map[int64]*migration.Migration)
	})
	dir := t.TempDir()

	tt := []struct {
		name         string
		typ          SourceType
		wantFilename string
		goldenFile   string
		opt          *CreateOptions
	}{
		{
			name:         "create users table",
			typ:          SourceTypeSQL,
			wantFilename: "00001_create_users_table.sql",
			goldenFile:   "testdata/create/sql.golden",
			opt:          &CreateOptions{Sequential: true, NoTx: false},
		},
		{

			name:         "add LOTS of users",
			typ:          SourceTypeSQL,
			wantFilename: "00002_add_lots_of_users.sql",
			goldenFile:   "testdata/create/sql_notx.golden",
			opt:          &CreateOptions{Sequential: true, NoTx: true},
		},
		{

			name:         "backfill emails",
			typ:          SourceTypeGo,
			wantFilename: "00003_backfill_emails.go",
			goldenFile:   "testdata/create/00003_backfill_emails.go.golden",
			opt:          &CreateOptions{Sequential: true, NoTx: false},
		},
		{

			name:         "insert a bunch of data",
			typ:          SourceTypeGo,
			wantFilename: "00004_insert_a_bunch_of_data.go",
			goldenFile:   "testdata/create/00004_insert_a_bunch_of_data.go.golden",
			opt:          &CreateOptions{Sequential: true, NoTx: true},
		},
	}
	for _, tc := range tt {
		filename, err := Create(dir, tc.typ, tc.name, tc.opt)
		check.NoError(t, err)
		check.Equal(t, filepath.Base(filename), tc.wantFilename)
		got := mustReadFile(t, filename)
		want := mustReadFile(t, tc.goldenFile)
		if strings.TrimSpace(got) != strings.TrimSpace(want) {
			fmt.Printf("got:\n%s\n", got)
			fmt.Println("---")
			fmt.Printf("want:\n%s\n", want)
			t.Errorf("expected output does not match, see log above")
		}
		if tc.typ == SourceTypeGo {
			// Must register the migration to avoid an error:
			// "go functions must be registered and built into a custom binary..."
			//
			// TODO: Can we avoid this by passing these into the Provider directly?
			err := register(filepath.Base(filename), tc.opt.NoTx, nil, nil, nil, nil)
			check.NoError(t, err)
		}
	}
	files, err := os.ReadDir(dir)
	check.NoError(t, err)
	// Check files are in order
	for i, f := range files {
		expected := fmt.Sprintf("%05v", i+1)
		if !strings.HasPrefix(f.Name(), expected) {
			t.Errorf("failed to find %s prefix in %s", expected, f.Name())
		}
	}
}

func TestTimestamped(t *testing.T) {
	t.Parallel()

	// We tested the template comonents in the sequential test, so we just need to
	// check that the timestamp is correct.

	dir := t.TempDir()

	tt := []struct {
		name         string
		typ          SourceType
		wantFilename string
	}{
		{
			name:         "create users table",
			typ:          SourceTypeSQL,
			wantFilename: "20000210231205_create_users_table.sql",
		},
		{

			name:         "add LOTS of users",
			typ:          SourceTypeSQL,
			wantFilename: "20010210231205_add_lots_of_users.sql",
		},
		{

			name:         "backfill emails",
			typ:          SourceTypeGo,
			wantFilename: "20020210231205_backfill_emails.go",
		},
		{

			name:         "insert a bunch of data",
			typ:          SourceTypeGo,
			wantFilename: "20030210231205_insert_a_bunch_of_data.go",
		},
	}
	for i, tc := range tt {
		// We need to use a fixed time for the test, but we also need to make sure
		// that the time is different for each test case, so we bump the year.
		now := time.Date(2000+i, time.February, 10, 23, 12, 5, 3, time.UTC)
		filename, err := Create(dir, tc.typ, tc.name, &CreateOptions{
			timeFunc: func() time.Time { return now },
		})
		check.NoError(t, err)
		check.Equal(t, filepath.Base(filename), tc.wantFilename)
	}
}

func TestCamelSnake(t *testing.T) {
	t.Parallel()

	tt := []struct {
		in    string
		camel string
		snake string
	}{
		{in: "Add updated_at to users table", camel: "AddUpdatedAtToUsersTable", snake: "add_updated_at_to_users_table"},
		{in: "$()&^%(_--crazy__--input$)", camel: "CrazyInput", snake: "crazy_input"},
	}

	for _, test := range tt {
		if got := camelCase(test.in); got != test.camel {
			t.Errorf("unexpected CamelCase for input(%q): got %q, want %q", test.in, got, test.camel)
		}
		if got := snakeCase(test.in); got != test.snake {
			t.Errorf("unexpected snake_case for input(%q): got %q, want %q", test.in, got, test.snake)
		}
	}
}

func mustReadFile(t *testing.T, filename string) string {
	got, err := os.ReadFile(filename)
	check.NoError(t, err)
	return string(got)
}
