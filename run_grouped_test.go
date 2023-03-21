package goose

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/migration"
)

func TestSplitMigrationsIntoGroups(t *testing.T) {
	tt := []struct {
		migrations []*migration.Migration
		expected   [][]int64
	}{
		{
			migrations: []*migration.Migration{},
			expected:   nil,
		},
		{
			migrations: []*migration.Migration{
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
			migrations: []*migration.Migration{
				newGo(3, true), newSQL(4, true),
			},
			expected: [][]int64{
				{3, 4},
			},
		},
		{
			migrations: []*migration.Migration{
				newGo(3, false),
				newSQL(4, false),
			},
			expected: [][]int64{
				{3},
				{4},
			},
		},
		{
			migrations: []*migration.Migration{
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
			migrations: []*migration.Migration{
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
			migrations: []*migration.Migration{
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
			migrations: []*migration.Migration{
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
				if m.Version != tc.expected[i][j] {
					t.Errorf("expected %d, got %d", tc.expected[i][j], m.Version)
				}
			}
		}
	}
}

func newSQL(version int64, useTx bool) *migration.Migration {
	return &migration.Migration{
		Type:     migration.TypeSQL,
		Version:  version,
		Fullpath: newRandomFilename(10) + ".sql",
		SQL: &migration.SQL{
			UseTx: useTx,
		},
	}
}

func newGo(version int64, useTx bool) *migration.Migration {
	return &migration.Migration{
		Type:     migration.TypeGo,
		Version:  version,
		Fullpath: newRandomFilename(10) + ".go",
		Go: &migration.Go{
			UseTx: useTx,
		},
	}
}

func newRandomFilename(n int) string {
	now := time.Now()
	version := now.Format(timestampFormat)
	return fmt.Sprintf("%v_%s", version, randString(n))
}

// randString generated a random lower case string of length n
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
