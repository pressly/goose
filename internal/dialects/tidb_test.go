package dialects_test

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3/database/dialect"
	"github.com/pressly/goose/v3/internal/dialects"
)

func TestTidbTableExists(t *testing.T) {
	var _ dialect.QuerierExtender = dialects.NewTidb()

	q := dialects.NewTidb().TableExists("goose_db_version")
	if q == "" {
		t.Fatal("expected non-empty TableExists query")
	}
	if !strings.Contains(q, "information_schema") {
		t.Fatalf("expected information_schema in query, got: %s", q)
	}
}

func TestTidbTableExistsWithSchema(t *testing.T) {
	q := dialects.NewTidb().TableExists("myschema.goose_db_version")
	if !strings.Contains(q, "myschema") {
		t.Fatalf("expected schema name in query, got: %s", q)
	}
	if !strings.Contains(q, "goose_db_version") {
		t.Fatalf("expected table name in query, got: %s", q)
	}
}
