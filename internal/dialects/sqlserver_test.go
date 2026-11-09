package dialects_test

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3/database/dialect"
	"github.com/pressly/goose/v3/internal/dialects"
)

func TestSqlserverTableExists(t *testing.T) {
	var _ dialect.QuerierExtender = dialects.NewSqlserver()

	q := dialects.NewSqlserver().TableExists("goose_db_version")
	if q == "" {
		t.Fatal("expected non-empty TableExists query")
	}
	if !strings.Contains(q, "information_schema") {
		t.Fatalf("expected information_schema in query, got: %s", q)
	}
	if !strings.Contains(q, "EXISTS") {
		t.Fatalf("expected EXISTS pattern in query for consistency, got: %s", q)
	}
}

func TestSqlserverTableExistsWithSchema(t *testing.T) {
	var _ dialect.QuerierExtender = dialects.NewSqlserver()

	q := dialects.NewSqlserver().TableExists("myschema.goose_db_version")
	if !strings.Contains(q, "myschema") {
		t.Fatalf("expected schema name in query, got: %s", q)
	}
	if !strings.Contains(q, "goose_db_version") {
		t.Fatalf("expected table name in query, got: %s", q)
	}
}
