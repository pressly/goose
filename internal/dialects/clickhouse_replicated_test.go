package dialects

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

// normalizeSQL collapses whitespace so tests can compare SQL strings without
// being sensitive to formatting.
func normalizeSQL(s string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
}

func assertSQL(t *testing.T, method, got, want string) {
	t.Helper()
	if normalizeSQL(got) != normalizeSQL(want) {
		t.Errorf("%s SQL mismatch\n got: %s\nwant: %s", method, normalizeSQL(got), normalizeSQL(want))
	}
}

const testTable = "goose_db_version"

func TestClickhouseReplicated_AllOptionsSet(t *testing.T) {
	q := NewClickhouseReplicated(
		WithClickhouseCluster("goose_cluster"),
		WithClickhouseZooKeeperPath("/clickhouse/tables/{shard}/goose_db_version"),
		WithClickhouseReplicaName("{replica}"),
		WithClickhouseInsertQuorum("3"),
	)

	assertSQL(t, "CreateTable", q.CreateTable(testTable), `
		CREATE TABLE IF NOT EXISTS goose_db_version ON CLUSTER goose_cluster (
			version_id Int64,
			is_applied UInt8,
			tstamp DateTime64(6) DEFAULT now64(6)
		)
		ENGINE = ReplicatedReplacingMergeTree('/clickhouse/tables/{shard}/goose_db_version', '{replica}', tstamp)
		ORDER BY (version_id)`)

	assertSQL(t, "InsertVersion", q.InsertVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) SETTINGS insert_quorum=3 VALUES ($1, $2)`)

	assertSQL(t, "DeleteVersion", q.DeleteVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) SETTINGS insert_quorum=3 VALUES ($1, 0)`)

	assertSQL(t, "GetMigrationByVersion", q.GetMigrationByVersion(testTable),
		`SELECT argMax(tstamp, tstamp) AS tstamp, argMax(is_applied, tstamp) AS is_applied FROM goose_db_version WHERE version_id = $1 GROUP BY version_id SETTINGS select_sequential_consistency=1`)

	assertSQL(t, "ListMigrations", q.ListMigrations(testTable),
		`SELECT version_id, is_applied FROM (
	SELECT version_id, argMax(is_applied, tstamp) AS is_applied FROM goose_db_version GROUP BY version_id
) WHERE is_applied = 1 ORDER BY version_id DESC SETTINGS select_sequential_consistency=1`)

	assertSQL(t, "GetLatestVersion", q.GetLatestVersion(testTable),
		`SELECT max(version_id) FROM (SELECT version_id, argMax(is_applied, tstamp) AS is_applied FROM goose_db_version GROUP BY version_id) WHERE is_applied = 1 SETTINGS select_sequential_consistency=1`)
}

func TestClickhouseReplicated_EnvDefaults(t *testing.T) {
	// Only the required env var; everything else should hit defaults.
	t.Setenv(EnvClickhouseCluster, "envcluster")
	t.Setenv(EnvClickhouseZKPath, "")
	t.Setenv(EnvClickhouseReplicaName, "")
	t.Setenv(EnvClickhouseInsertQuorum, "")

	q := NewClickhouseReplicated()

	assertSQL(t, "CreateTable", q.CreateTable(testTable), `
		CREATE TABLE IF NOT EXISTS goose_db_version ON CLUSTER envcluster (
			version_id Int64,
			is_applied UInt8,
			tstamp DateTime64(6) DEFAULT now64(6)
		)
		ENGINE = ReplicatedReplacingMergeTree(tstamp)
		ORDER BY (version_id)`)

	assertSQL(t, "InsertVersion", q.InsertVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) SETTINGS insert_quorum='auto' VALUES ($1, $2)`)

	assertSQL(t, "DeleteVersion", q.DeleteVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) SETTINGS insert_quorum='auto' VALUES ($1, 0)`)
}

func TestClickhouseReplicated_EnvOverridesAllApplied(t *testing.T) {
	t.Setenv(EnvClickhouseCluster, "envcluster")
	t.Setenv(EnvClickhouseZKPath, "/zk/env")
	t.Setenv(EnvClickhouseReplicaName, "env-replica")
	t.Setenv(EnvClickhouseInsertQuorum, "2")

	q := NewClickhouseReplicated()

	assertSQL(t, "CreateTable", q.CreateTable(testTable), `
		CREATE TABLE IF NOT EXISTS goose_db_version ON CLUSTER envcluster (
			version_id Int64,
			is_applied UInt8,
			tstamp DateTime64(6) DEFAULT now64(6)
		)
		ENGINE = ReplicatedReplacingMergeTree('/zk/env', 'env-replica', tstamp)
		ORDER BY (version_id)`)

	assertSQL(t, "InsertVersion", q.InsertVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) SETTINGS insert_quorum=2 VALUES ($1, $2)`)

	assertSQL(t, "DeleteVersion", q.DeleteVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) SETTINGS insert_quorum=2 VALUES ($1, 0)`)
}

func TestClickhouseReplicated_OptionsOverrideEnv(t *testing.T) {
	t.Setenv(EnvClickhouseCluster, "envcluster")
	t.Setenv(EnvClickhouseInsertQuorum, "5")

	q := NewClickhouseReplicated(
		WithClickhouseCluster("optcluster"),
		WithClickhouseInsertQuorum("1"),
	)

	if got := q.CreateTable(testTable); !strings.Contains(got, "ON CLUSTER optcluster") {
		t.Errorf("expected option to override env cluster; got: %s", got)
	}
	if got := q.InsertVersion(testTable); !strings.Contains(got, "insert_quorum=1") {
		t.Errorf("expected option to override env insert_quorum on InsertVersion; got: %s", got)
	}
	if got := q.DeleteVersion(testTable); !strings.Contains(got, "insert_quorum=1") {
		t.Errorf("expected option to override env insert_quorum on DeleteVersion; got: %s", got)
	}
}

func TestClickhouseReplicated_MissingClusterPanics(t *testing.T) {
	t.Setenv(EnvClickhouseCluster, "")

	q := NewClickhouseReplicated()

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on empty cluster")
		}
		err, ok := r.(error)
		if !ok {
			t.Fatalf("expected panic to carry an error, got %T: %v", r, r)
		}
		if !errors.Is(err, ErrClickhouseReplicatedNoCluster) {
			t.Errorf("expected ErrClickhouseReplicatedNoCluster, got: %v", err)
		}
		msg := err.Error()
		if !strings.Contains(msg, EnvClickhouseCluster) {
			t.Errorf("expected error to mention %q, got: %s", EnvClickhouseCluster, msg)
		}
		if !strings.Contains(msg, "WithClickhouseCluster") {
			t.Errorf("expected error to mention WithClickhouseCluster, got: %s", msg)
		}
	}()

	_ = q.CreateTable(testTable)
}

// TestClickhouseReplicated_QuorumQuoting verifies quorum formatting on both
// the up (InsertVersion) and down (DeleteVersion) writes: numeric values are
// emitted bare, symbolic values ("auto", "off") are single-quoted.
func TestClickhouseReplicated_QuorumQuoting(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"auto", "'auto'"},
		{"off", "'off'"},
		{"3", "3"},
		{"0", "0"},
	}
	for _, tc := range cases {
		q := NewClickhouseReplicated(
			WithClickhouseCluster("c"),
			WithClickhouseInsertQuorum(tc.in),
		)
		for _, sql := range []struct {
			name, got string
		}{
			{"InsertVersion", q.InsertVersion(testTable)},
			{"DeleteVersion", q.DeleteVersion(testTable)},
		} {
			if !strings.Contains(sql.got, "insert_quorum="+tc.want) {
				t.Errorf("%s quorum %q: expected insert_quorum=%s in SQL, got: %s", sql.name, tc.in, tc.want, sql.got)
			}
		}
	}
}
