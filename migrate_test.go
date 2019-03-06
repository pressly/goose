package goose

import (
	"testing"
)

func TestMigrationSort(t *testing.T) {
	t.Parallel()

	ms := Migrations{}

	// insert in any order
	ms = append(ms, newMigration(20120000, "test"))
	ms = append(ms, newMigration(20128000, "test"))
	ms = append(ms, newMigration(20129000, "test"))
	ms = append(ms, newMigration(20127000, "test"))

	ms = sortAndConnectMigrations(ms)

	sorted := []int64{20120000, 20127000, 20128000, 20129000}

	validateMigrationSort(t, ms, sorted)
}

func newMigration(v int64, src string) *Migration {
	return &Migration{Version: v, Previous: -1, Next: -1, Source: src}
}

func validateMigrationSort(t *testing.T, ms Migrations, sorted []int64) {
	for i, m := range ms {
		if sorted[i] != m.Version {
			t.Error("incorrect sorted version")
		}

		var next, prev int64

		if i == 0 {
			prev = -1
			next = ms[i+1].Version
		} else if i == len(ms)-1 {
			prev = ms[i-1].Version
			next = -1
		} else {
			prev = ms[i-1].Version
			next = ms[i+1].Version
		}

		if m.Next != next {
			t.Errorf("mismatched Next. v: %v, got %v, wanted %v\n", m, m.Next, next)
		}

		if m.Previous != prev {
			t.Errorf("mismatched Previous v: %v, got %v, wanted %v\n", m, m.Previous, prev)
		}
	}

	t.Log(ms)
}
