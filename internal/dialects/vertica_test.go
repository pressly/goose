package dialects_test

import (
	"strings"
	"testing"

	"github.com/pressly/goose/v3/database/dialect"
	"github.com/pressly/goose/v3/internal/dialects"
)

func TestVerticaTableExists(t *testing.T) {
	var _ dialect.QuerierExtender = dialects.NewVertica()

	q := dialects.NewVertica().TableExists("goose_db_version")
	if q == "" {
		t.Fatal("expected non-empty TableExists query")
	}
	if !strings.Contains(q, "v_catalog") {
		t.Fatalf("expected v_catalog in query, got: %s", q)
	}
}

func TestVerticaTableExistsWithSchema(t *testing.T) {
	q := dialects.NewVertica().TableExists("myschema.goose_db_version")
	if !strings.Contains(q, "myschema") {
		t.Fatalf("expected schema name in query, got: %s", q)
	}
	if !strings.Contains(q, "goose_db_version") {
		t.Fatalf("expected table name in query, got: %s", q)
	}
}
