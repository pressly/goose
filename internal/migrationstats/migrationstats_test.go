package migrationstats

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v4/internal/check"
)

func TestParsingGoMigrations(t *testing.T) {
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

func TestParsingGoMigrationsError(t *testing.T) {
	_, err := parseGoFile(strings.NewReader(emptyInit))
	check.HasError(t, err)
	check.Contains(t, err.Error(), "no registered goose functions")

	_, err = parseGoFile(strings.NewReader(wrongName))
	check.HasError(t, err)
	check.Contains(t, err.Error(), "AddMigration or AddMigrationNoTx")
}

var (
	upAndDown = `package foo

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(up001, down001)
}

func up001(ctx context.Context, tx *sql.Tx) error { return nil }

func down001(ctx context.Context, tx *sql.Tx) error { return nil }`

	downOnly = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(nil, down002)
}

func down002(ctx context.Context, tx *sql.Tx) error { return nil }`

	upOnly = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(up003, nil)
}

func up003(ctx context.Context, tx *sql.Tx) error { return nil }`

	upAndDownNil = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigration(nil, nil)
}`
)
var (
	upAndDownNoTx = `package foo

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationNoTx(up001, down001)
}

func up001(ctx context.Context, db *sql.DB) error { return nil }

func down001(ctx context.Context, db *sql.DB) error { return nil }`

	downOnlyNoTx = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationNoTx(nil, down002)
}

func down002(ctx context.Context, db *sql.DB) error { return nil }`

	upOnlyNoTx = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationNoTx(up003, nil)
}

func up003(ctx context.Context, db *sql.DB) error { return nil }`

	upAndDownNilNoTx = `package testgo

import (
	"database/sql"

	"github.com/pressly/goose/v4"
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

	"github.com/pressly/goose/v4"
)

func init() {
	goose.AddMigrationWrongName(nil, nil)
}`
)
