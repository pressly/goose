package goose

import (
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
		// 00003 is missing
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

	res, err := Fix(dir)
	check.NoError(t, err)
	check.Number(t, len(res), 4)

	want := []struct {
		wantOld string
		wantNew string
	}{
		{
			wantOld: "20000210231205_create_users_table.sql",
			wantNew: "00005_create_users_table.sql",
		},
		{
			wantOld: "20010210231205_add_lots_of_users.sql",
			wantNew: "00006_add_lots_of_users.sql",
		},
		{
			wantOld: "20020210231205_backfill_emails.go",
			wantNew: "00007_backfill_emails.go",
		},
		{
			wantOld: "20030210231205_insert_a_bunch_of_data.go",
			wantNew: "00008_insert_a_bunch_of_data.go",
		},
	}

	for i := range res {
		wantOld := filepath.Join(dir, want[i].wantOld)
		wantNew := filepath.Join(dir, want[i].wantNew)
		check.Equal(t, res[i].OldPath, wantOld)
		check.Equal(t, res[i].NewPath, wantNew)
	}
}
