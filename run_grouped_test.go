package goose

import (
	"testing"

	"github.com/pressly/goose/v4/internal/check"
)

func TestSplitMigrationsIntoGroups(t *testing.T) {
	tt := []struct {
		migrations []*migration
		expected   [][]int64
	}{
		{
			migrations: []*migration{},
			expected:   nil,
		},
		{
			migrations: []*migration{
				newSQL(11, true), newGo(12, true), newSQL(13, true),
				newSQL(14, false),
				newGo(15, true), newSQL(16, true),
			},
			expected: [][]int64{
				{11, 12, 13},
				{14},
				{15, 16},
			},
		},
		{
			migrations: []*migration{
				newGo(3, true), newSQL(4, true),
			},
			expected: [][]int64{
				{3, 4},
			},
		},
		{
			migrations: []*migration{
				newGo(3, false),
				newSQL(4, false),
			},
			expected: [][]int64{
				{3},
				{4},
			},
		},
		{
			migrations: []*migration{
				newGo(3, false),
				newSQL(4, true),
				newSQL(5, false),
			},
			expected: [][]int64{
				{3},
				{4},
				{5},
			},
		},
		{
			migrations: []*migration{
				newGo(3, true),
				newSQL(4, false),
				newSQL(5, true),
			},
			expected: [][]int64{
				{3},
				{4},
				{5},
			},
		},
		{
			migrations: []*migration{
				newSQL(3, true), newSQL(4, true),
				newSQL(5, false),
				newGo(6, false),
			},
			expected: [][]int64{
				{3, 4},
				{5},
				{6},
			},
		},
		{
			migrations: []*migration{
				newSQL(3, true), newSQL(4, true),
				newSQL(5, false),
				newGo(6, false),
				newSQL(7, true),
			},
			expected: [][]int64{
				{3, 4},
				{5},
				{6},
				{7},
			},
		},
	}

	for _, tc := range tt {
		groups := splitMigrationsIntoGroups(tc.migrations)
		check.Number(t, len(groups), len(tc.expected))
		for i, g := range groups {
			check.Number(t, len(g), len(tc.expected[i]))
			for j, m := range g {
				if m.version != tc.expected[i][j] {
					t.Errorf("expected %d, got %d", tc.expected[i][j], m.version)
				}
			}
		}
	}
}

func newSQL(version int64, useTx bool) *migration {
	return &migration{
		migrationType: MigrationTypeSQL,
		sqlMigration:  &sqlMigration{useTx: useTx},
		version:       version,
	}
}

func newGo(version int64, useTx bool) *migration {
	return &migration{
		migrationType: MigrationTypeGo,
		goMigration:   &goMigration{useTx: useTx},
		version:       version,
	}
}
