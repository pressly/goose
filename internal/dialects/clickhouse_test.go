package dialects_test

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3/database/dialect"
	"github.com/pressly/goose/v3/internal/dialects"
)

func TestClickhouseTableExists(t *testing.T) {
	var _ dialect.QuerierExtender = dialects.NewClickhouse()

	q := dialects.NewClickhouse().TableExists("goose_db_version")
	if q == "" {
		t.Fatal("expected non-empty TableExists query")
	}
	if !strings.Contains(q, "system.tables") {
		t.Fatalf("expected system.tables in query, got: %s", q)
	}
}

func TestClickhouseTableExistsWithSchema(t *testing.T) {
	q := dialects.NewClickhouse().TableExists("mydb.goose_db_version")
	if !strings.Contains(q, "mydb") {
		t.Fatalf("expected database name in query, got: %s", q)
	}
	if !strings.Contains(q, "goose_db_version") {
		t.Fatalf("expected table name in query, got: %s", q)
	}
}
