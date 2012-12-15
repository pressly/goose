package main

import (
	"testing"
)

func TestMigrationMapSortUp(t *testing.T) {

	mm := &MigrationMap{
		Migrations: make(map[int]Migration),
	}

	// insert in any order
	mm.Append(20120000, "test")
	mm.Append(20128000, "test")
	mm.Append(20129000, "test")
	mm.Append(20127000, "test")

	mm.Sort(true) // sort Upwards

	sorted := []int{20120000, 20127000, 20128000, 20129000}

	validateMigrationMapIsSorted(t, mm, sorted)
}

func TestMigrationMapSortDown(t *testing.T) {

	mm := &MigrationMap{
		Migrations: make(map[int]Migration),
	}

	// insert in any order
	mm.Append(20120000, "test")
	mm.Append(20128000, "test")
	mm.Append(20129000, "test")
	mm.Append(20127000, "test")

	mm.Sort(false) // sort Downwards

	sorted := []int{20129000, 20128000, 20127000, 20120000}

	validateMigrationMapIsSorted(t, mm, sorted)
}

func validateMigrationMapIsSorted(t *testing.T, mm *MigrationMap, sorted []int) {

	for i, v := range mm.Versions {
		if sorted[i] != v {
			t.Error("incorrect sorted version")
		}

		var next, prev int

		if i == 0 {
			prev = -1
			next = mm.Versions[i+1]
		} else if i == len(mm.Versions)-1 {
			prev = mm.Versions[i-1]
			next = -1
		} else {
			prev = mm.Versions[i-1]
			next = mm.Versions[i+1]
		}

		if mm.Migrations[v].Next != next {
			t.Errorf("mismatched Next. v: %v, got %v, wanted %v\n", v, mm.Migrations[v].Next, next)
		}

		if mm.Migrations[v].Previous != prev {
			t.Errorf("mismatched Previous v: %v, got %v, wanted %v\n", v, mm.Migrations[v].Previous, prev)
		}
	}
}
