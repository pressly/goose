package dialects_test

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3/database/dialect"
	"github.com/pressly/goose/v3/internal/dialects"
)

func TestSpannerTableExists(t *testing.T) {
	var _ dialect.QuerierExtender = dialects.NewSpanner()

	q := dialects.NewSpanner().TableExists("goose_db_version")
	if q == "" {
		t.Fatal("expected non-empty TableExists query")
	}
	if !strings.Contains(q, "information_schema") {
		t.Fatalf("expected information_schema in query, got: %s", q)
	}
}

func TestSpannerTableExistsDefaultSchema(t *testing.T) {
	// Spanner's default schema is an empty string; unqualified table names
	// should produce a query with table_schema = '' (empty literal).
	q := dialects.NewSpanner().TableExists("goose_db_version")
	if !strings.Contains(q, "goose_db_version") {
		t.Fatalf("expected table name in query, got: %s", q)
	}
}

func TestSpannerTableExistsWithSchema(t *testing.T) {
	q := dialects.NewSpanner().TableExists("myschema.goose_db_version")
	if !strings.Contains(q, "myschema") {
		t.Fatalf("expected schema name in query, got: %s", q)
	}
}
