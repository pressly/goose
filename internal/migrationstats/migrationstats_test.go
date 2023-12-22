package migrationstats

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pressly/goose/v3/internal/check"
)

func TestParsingGoMigrations(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name                     string
		input                    string
		wantUpName, wantDownName string
		wantTx                   bool
	}{
		// AddMigration
		{"upAndDown", upAndDown, "up001", "down001", true},
		{"downOnly", downOnly, "nil", "down002", true},
		{"upOnly", upOnly, "up003", "nil", true},
		{"upAndDownNil", upAndDownNil, "nil", "nil", true},
		// AddMigrationNoTx
		{"upAndDownNoTx", upAndDownNoTx, "up001", "down001", false},
		{"downOnlyNoTx", downOnlyNoTx, "nil", "down002", false},
		{"upOnlyNoTx", upOnlyNoTx, "up003", "nil", false},
		{"upAndDownNilNoTx", upAndDownNilNoTx, "nil", "nil", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, err := parseGoFile(strings.NewReader(tc.input))
			check.NoError(t, err)
			check.Equal(t, g.useTx != nil, true)
			check.Bool(t, *g.useTx, tc.wantTx)
			check.Equal(t, g.downFuncName, tc.wantDownName)
			check.Equal(t, g.upFuncName, tc.wantUpName)
		})
	}
}

func TestGoMigrationStats(t *testing.T) {
	t.Parallel()

	base := "../../tests/gomigrations/success/testdata"
	all, err := os.ReadDir(base)
	check.NoError(t, err)
	check.Equal(t, len(all), 16)
	files := make([]string, 0, len(all))
	for _, f := range all {
		files = append(files, filepath.Join(base, f.Name()))
	}
	stats, err := GatherStats(NewFileWalker(files...), false)
	check.NoError(t, err)
	check.Equal(t, len(stats), 16)
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
	check.Equal(t, filepath.Base(stats.FileName), filename)
	check.Equal(t, stats.Version, version)
	check.Equal(t, stats.UpCount, upCount)
	check.Equal(t, stats.DownCount, downCount)
	check.Equal(t, stats.Tx, tx)
}

func TestParsingGoMigrationsError(t *testing.T) {
	t.Parallel()
	_, err := parseGoFile(strings.NewReader(emptyInit))
	check.HasError(t, err)
	check.Contains(t, err.Error(), "no registered goose functions")

	_, err = parseGoFile(strings.NewReader(wrongName))
	check.HasError(t, err)
	check.Contains(t, err.Error(), "AddMigration, AddMigrationNoTx, AddMigrationContext, AddMigrationNoTxContext")
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
)
