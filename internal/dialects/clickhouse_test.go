package dialects

import (
	"strings"
	"testing"
)

// TestClickhouse_TableExists_EngineGuard verifies the TableExists query embeds a throwIf guard
// that raises a clear error if the table exists with an engine other than MergeTree (i.e. it was
// created by the clickhouse-replicated dialect instead).
func TestClickhouse_TableExists_EngineGuard(t *testing.T) {
	q := NewClickhouse()
	querier, ok := q.(interface{ TableExists(string) string })
	if !ok {
		t.Fatal("querier does not implement TableExists")
	}
	got := querier.TableExists(testTable)
	for _, want := range []string{
		"system.tables",
		"throwIf(",
		"'MergeTree'",
		"the clickhouse-replicated dialect",
		testTable,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("TableExists SQL missing %q\ngot: %s", want, got)
		}
	}
}
