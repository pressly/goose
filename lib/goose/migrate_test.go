package main

import (
	"testing"
)

func TestMigrationMapSortUp(t *testing.T) {

	mm := &MigrationMap{}

	// insert in any order
	mm.Append(20120000, "test")
	mm.Append(20128000, "test")
	mm.Append(20129000, "test")
	mm.Append(20127000, "test")

	mm.Sort(true) // sort Upwards

	sorted := []int64{20120000, 20127000, 20128000, 20129000}

	validateMigrationMapIsSorted(t, mm, sorted)
}

func TestMigrationMapSortDown(t *testing.T) {

	mm := &MigrationMap{}

	// insert in any order
	mm.Append(20120000, "test")
	mm.Append(20128000, "test")
	mm.Append(20129000, "test")
	mm.Append(20127000, "test")

	mm.Sort(false) // sort Downwards

	sorted := []int64{20129000, 20128000, 20127000, 20120000}

	validateMigrationMapIsSorted(t, mm, sorted)
}

func validateMigrationMapIsSorted(t *testing.T, mm *MigrationMap, sorted []int64) {

	for i, m := range mm.Migrations {
		if sorted[i] != m.Version {
			t.Error("incorrect sorted version")
		}

		var next, prev int64

		if i == 0 {
			prev = -1
			next = mm.Migrations[i+1].Version
		} else if i == len(mm.Migrations)-1 {
			prev = mm.Migrations[i-1].Version
			next = -1
		} else {
			prev = mm.Migrations[i-1].Version
			next = mm.Migrations[i+1].Version
		}

		if m.Next != next {
			t.Errorf("mismatched Next. v: %v, got %v, wanted %v\n", m, m.Next, next)
		}

		if m.Previous != prev {
			t.Errorf("mismatched Previous v: %v, got %v, wanted %v\n", m, m.Previous, prev)
		}
	}

	t.Log(mm.Migrations)
}
