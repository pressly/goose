package migrationstats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsingGoMigrations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                   string
		input                  string
		wantUpNil, wantDownNil bool
		wantTx                 bool
	}{
		// AddMigration
		{"upAndDown", upAndDown, false, false, true},
		{"downOnly", downOnly, true, false, true},
		{"upOnly", upOnly, false, true, true},
		{"upAndDownNil", upAndDownNil, true, true, true},
		// AddMigrationNoTx
		{"upAndDownNoTx", upAndDownNoTx, false, false, false},
		{"downOnlyNoTx", downOnlyNoTx, true, false, false},
		{"upOnlyNoTx", upOnlyNoTx, false, true, false},
		{"upAndDownNilNoTx", upAndDownNilNoTx, true, true, false},
		// Inlined function literals, see https://github.com/pressly/goose/issues/519
		{"upAndDownInline", upAndDownInline, false, false, true},
		{"upInlineDownNil", upInlineDownNil, false, true, true},
		{"upNilDownInline", upNilDownInline, true, false, true},
		{"upAndDownInlineNoTx", upAndDownInlineNoTx, false, false, false},
		{"upAndDownInlineContext", upAndDownInlineContext, false, false, true},
		{"upAndDownInlineNoTxContext", upAndDownInlineNoTxContext, false, false, false},
		{"upInlineDownNamed", upInlineDownNamed, false, false, true},
		// Functions referenced through another package
		{"upAndDownQualified", upAndDownQualified, false, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, err := parseGoFile(strings.NewReader(tc.input))
			require.NoError(t, err)
			require.NotNil(t, g.useTx)
			require.Equal(t, tc.wantTx, *g.useTx)
			require.Equal(t, tc.wantDownNil, g.downFuncNil)
			require.Equal(t, tc.wantUpNil, g.upFuncNil)
		})
	}
}

func TestGoMigrationStatsInline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		input            string
		wantUp, wantDown int
		wantTx           bool
	}{
		{"upAndDownInline", upAndDownInline, 1, 1, true},
		{"upInlineDownNil", upInlineDownNil, 1, 0, true},
		{"upNilDownInline", upNilDownInline, 0, 1, true},
		{"upAndDownInlineNoTx", upAndDownInlineNoTx, 1, 1, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			filename := filepath.Join(dir, "001_"+tc.name+".go")
			require.NoError(t, os.WriteFile(filename, []byte(tc.input), 0o644))
			stats, err := GatherStats(NewFileWalker(filename), false)
			require.NoError(t, err)
			require.Len(t, stats, 1)
			checkGoStats(t, stats[0], filepath.Base(filename), 1, tc.wantUp, tc.wantDown, tc.wantTx)
		})
	}
}

func TestGoMigrationStats(t *testing.T) {
	t.Parallel()

	base := "../../tests/gomigrations/success/testdata"
	all, err := os.ReadDir(base)
	require.NoError(t, err)
	require.Len(t, all, 16)
	files := make([]string, 0, len(all))
	for _, f := range all {
		files = append(files, filepath.Join(base, f.Name()))
	}
	stats, err := GatherStats(NewFileWalker(files...), false)
	require.NoError(t, err)
	require.Len(t, stats, 16)
	checkGoStats(t, stats[0], "001_up_down.go", 1, 1, 1, true)
	checkGoStats(t, stats[1], "002_up_only.go", 2, 1, 0, true)
	checkGoStats(t, stats[2], "003_down_only.go", 3, 0, 1, true)
	checkGoStats(t, stats[3], "004_empty.go", 4, 0, 0, true)
	checkGoStats(t, stats[4], "005_up_down_no_tx.go", 5, 1, 1, false)
	checkGoStats(t, stats[5], "006_up_only_no_tx.go", 6, 1, 0, false)
	checkGoStats(t, stats[6], "007_down_only_no_tx.go", 7, 0, 1, false)
	checkGoStats(t, stats[7], "008_empty_no_tx.go", 8, 0, 0, false)
	checkGoStats(t, stats[8], "009_up_down_ctx.go", 9, 1, 1, true)
	checkGoStats(t, stats[9], "010_up_only_ctx.go", 10, 1, 0, true)
	checkGoStats(t, stats[10], "011_down_only_ctx.go", 11, 0, 1, true)
	checkGoStats(t, stats[11], "012_empty_ctx.go", 12, 0, 0, true)
	checkGoStats(t, stats[12], "013_up_down_no_tx_ctx.go", 13, 1, 1, false)
	checkGoStats(t, stats[13], "014_up_only_no_tx_ctx.go", 14, 1, 0, false)
	checkGoStats(t, stats[14], "015_down_only_no_tx_ctx.go", 15, 0, 1, false)
	checkGoStats(t, stats[15], "016_empty_no_tx_ctx.go", 16, 0, 0, false)
}

func checkGoStats(t *testing.T, stats *Stats, filename string, version int64, upCount, downCount int, tx bool) {
	t.Helper()
	require.Equal(t, filepath.Base(stats.FileName), filename)
	require.Equal(t, stats.Version, version)
	require.Equal(t, stats.UpCount, upCount)
	require.Equal(t, stats.DownCount, downCount)
	require.Equal(t, stats.Tx, tx)
}

func TestParsingGoMigrationsError(t *testing.T) {
	t.Parallel()
	_, err := parseGoFile(strings.NewReader(emptyInit))
	require.Error(t, err)
	require.Contains(t, err.Error(), "no registered goose functions")

	_, err = parseGoFile(strings.NewReader(wrongName))
	require.Error(t, err)
	require.Contains(t, err.Error(), "AddMigration, AddMigrationNoTx, AddMigrationContext, AddMigrationNoTxContext")

	_, err = parseGoFile(strings.NewReader(wrongArgCount))
	require.Error(t, err)
	require.Contains(t, err.Error(), "registered goose functions have 2 arguments: got 1")
}

var (
	upAndDown = `package foo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(up001, down001)
}

func up001(tx *sql.Tx) error { return nil }

func down001(tx *sql.Tx) error { return nil }`

	downOnly = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(nil, down002)
}

func down002(tx *sql.Tx) error { return nil }`

	upOnly = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(up003, nil)
}

func up003(tx *sql.Tx) error { return nil }`

	upAndDownNil = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(nil, nil)
}`
)
var (
	upAndDownNoTx = `package foo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(up001, down001)
}

func up001(db *sql.DB) error { return nil }

func down001(db *sql.DB) error { return nil }`

	downOnlyNoTx = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(nil, down002)
}

func down002(db *sql.DB) error { return nil }`

	upOnlyNoTx = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(up003, nil)
}

func up003(db *sql.DB) error { return nil }`

	upAndDownNilNoTx = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(nil, nil)
}`
)

var (
	emptyInit = `package testgo

func init() {}`

	wrongName = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationWrongName(nil, nil)
}`

	wrongArgCount = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(nil)
}`
)

var (
	upAndDownInline = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(func(tx *sql.Tx) error { return nil }, func(tx *sql.Tx) error { return nil })
}`

	upInlineDownNil = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(func(tx *sql.Tx) error { return nil }, nil)
}`

	upNilDownInline = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(nil, func(tx *sql.Tx) error { return nil })
}`

	upAndDownInlineNoTx = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTx(func(db *sql.DB) error { return nil }, func(db *sql.DB) error { return nil })
}`

	upAndDownInlineContext = `package testgo

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationContext(
		func(ctx context.Context, tx *sql.Tx) error { return nil },
		func(ctx context.Context, tx *sql.Tx) error { return nil },
	)
}`

	upAndDownInlineNoTxContext = `package testgo

import (
	"context"
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigrationNoTxContext(
		func(ctx context.Context, db *sql.DB) error { return nil },
		func(ctx context.Context, db *sql.DB) error { return nil },
	)
}`

	upInlineDownNamed = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v3"
)

func init() {
	goose.AddMigration(func(tx *sql.Tx) error { return nil }, down004)
}

func down004(tx *sql.Tx) error { return nil }`

	upAndDownQualified = `package testgo

import (
	"github.com/pressly/goose/v3"

	"example.com/migrations"
)

func init() {
	goose.AddMigration(migrations.Up005, migrations.Down005)
}`
)
