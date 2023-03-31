package goose

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v4/internal/check"
)

func TestFix(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	// Seed directory with some migrations.
	seed := []string{
		"00001_create_users_table.sql",
		"00002_add_lots_of_users.sql",
		"00003_backfill_emails.go",
		"00004_insert_a_bunch_of_data.go",
		"20000210231205_create_users_table.sql",
		"20010210231205_add_lots_of_users.sql",
		"20020210231205_backfill_emails.go",
		"20030210231205_insert_a_bunch_of_data.go",
	}
	for _, name := range seed {
		_, err := os.Create(filepath.Join(dir, name))
		check.NoError(t, err)
		if filepath.Ext(name) == ".go" {
			err = register(filepath.Join(dir, name), false, nil, nil, nil, nil)
			check.NoError(t, err)
		}
	}

	out, err := Fix(dir)
	check.NoError(t, err)
	fmt.Println(out)
}
