package dialects_test

import (
	"testing"

	"github.com/pressly/goose/v3/database/dialect"
	"github.com/pressly/goose/v3/internal/dialects"
)

func TestSqlite3TableExists(t *testing.T) {
	// compile-time: both must satisfy QuerierExtender
	var _ dialect.QuerierExtender = dialects.NewSqlite3()
	var _ dialect.QuerierExtender = dialects.NewTurso()

	q := dialects.NewSqlite3().TableExists("goose_db_version")
	if q == "" {
		t.Fatal("expected non-empty TableExists query for sqlite3")
	}
	tq := dialects.NewTurso().TableExists("goose_db_version")
	if tq == "" {
		t.Fatal("expected non-empty TableExists query for turso")
	}
}
