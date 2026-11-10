package dialects_test

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3/database/dialect"
	"github.com/pressly/goose/v3/internal/dialects"
)

func TestRedshiftTableExists(t *testing.T) {
	var _ dialect.QuerierExtender = dialects.NewRedshift()

	q := dialects.NewRedshift().TableExists("goose_db_version")
	if q == "" {
		t.Fatal("expected non-empty TableExists query")
	}
	if !strings.Contains(q, "pg_tables") {
		t.Fatalf("expected pg_tables in query, got: %s", q)
	}
}

func TestRedshiftTableExistsWithSchema(t *testing.T) {
	q := dialects.NewRedshift().TableExists("myschema.goose_db_version")
	if !strings.Contains(q, "myschema") {
		t.Fatalf("expected schema name in query, got: %s", q)
	}
	if !strings.Contains(q, "goose_db_version") {
		t.Fatalf("expected table name in query, got: %s", q)
	}
}
