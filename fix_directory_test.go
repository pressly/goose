package goose_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
)

func TestFixPreservesTimestampInDirectory(t *testing.T) {
	for _, tc := range []struct{ name, dir string }{
		{"plain", "migrations"},
		{"exact_timestamp", "20260102030405"},
		{"embedded_timestamp", "backup_20260102030405_migrations"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			dir := filepath.Join(root, tc.dir)
			if err := os.Mkdir(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			files := map[string]string{
				"00003_existing.sql":        "-- +goose Up\nSELECT 3;\n",
				"20260102030405_first.sql":  "-- +goose Up\nSELECT 4;\n",
				"20260102030406_second.sql": "-- +goose Up\nSELECT 5;\n",
			}
			for name, data := range files {
				if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := goose.Fix(dir); err != nil {
				t.Fatal(err)
			}
			for original, expected := range map[string]string{
				"00003_existing.sql":        "00003_existing.sql",
				"20260102030405_first.sql":  "00004_first.sql",
				"20260102030406_second.sql": "00005_second.sql",
			} {
				data, err := os.ReadFile(filepath.Join(dir, expected))
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != files[original] {
					t.Errorf("%s: contents changed", expected)
				}
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 3 {
				t.Errorf("got %d entries, want 3", len(entries))
			}
			if err := goose.Fix(dir); err != nil {
				t.Fatalf("second Fix: %v", err)
			}
		})
	}
}
