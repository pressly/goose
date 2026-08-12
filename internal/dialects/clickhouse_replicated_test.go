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
		WithClickhouseMutationsSync(1),
		WithClickhouseDeleteOnCluster(true),
	)

	assertSQL(t, "CreateTable", q.CreateTable(testTable), `
		CREATE TABLE IF NOT EXISTS goose_db_version ON CLUSTER goose_cluster (
			version_id Int64,
			is_applied UInt8,
			tstamp DateTime64(6) DEFAULT now64(6)
		)
		ENGINE = ReplicatedMergeTree('/clickhouse/tables/{shard}/goose_db_version', '{replica}')
		ORDER BY (version_id)`)

	assertSQL(t, "InsertVersion", q.InsertVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, $2) SETTINGS insert_quorum=3, select_sequential_consistency=1`)

	assertSQL(t, "DeleteVersion", q.DeleteVersion(testTable),
		`ALTER TABLE goose_db_version ON CLUSTER goose_cluster DELETE WHERE version_id = $1 SETTINGS mutations_sync = 1`)

	assertSQL(t, "GetMigrationByVersion", q.GetMigrationByVersion(testTable),
		`SELECT tstamp, is_applied FROM goose_db_version WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1`)

	assertSQL(t, "ListMigrations", q.ListMigrations(testTable),
		`SELECT version_id, is_applied FROM goose_db_version ORDER BY version_id DESC`)

	assertSQL(t, "GetLatestVersion", q.GetLatestVersion(testTable),
		`SELECT max(version_id) FROM goose_db_version`)
}

func TestClickhouseReplicated_EnvDefaults(t *testing.T) {
	// Only the required env var; everything else should hit defaults.
	t.Setenv(EnvClickhouseCluster, "envcluster")
	t.Setenv(EnvClickhouseZKPath, "")
	t.Setenv(EnvClickhouseReplicaName, "")
	t.Setenv(EnvClickhouseInsertQuorum, "")
	t.Setenv(EnvClickhouseMutationsSync, "")
	t.Setenv(EnvClickhouseDeleteOnCluster, "")

	q := NewClickhouseReplicated()

	assertSQL(t, "CreateTable", q.CreateTable(testTable), `
		CREATE TABLE IF NOT EXISTS goose_db_version ON CLUSTER envcluster (
			version_id Int64,
			is_applied UInt8,
			tstamp DateTime64(6) DEFAULT now64(6)
		)
		ENGINE = ReplicatedMergeTree
		ORDER BY (version_id)`)

	assertSQL(t, "InsertVersion", q.InsertVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, $2) SETTINGS insert_quorum='auto', select_sequential_consistency=1`)

	assertSQL(t, "DeleteVersion", q.DeleteVersion(testTable),
		`ALTER TABLE goose_db_version DELETE WHERE version_id = $1 SETTINGS mutations_sync = 2`)
}

func TestClickhouseReplicated_EnvOverridesAllApplied(t *testing.T) {
	t.Setenv(EnvClickhouseCluster, "envcluster")
	t.Setenv(EnvClickhouseZKPath, "/zk/env")
	t.Setenv(EnvClickhouseReplicaName, "env-replica")
	t.Setenv(EnvClickhouseInsertQuorum, "2")
	t.Setenv(EnvClickhouseMutationsSync, "0")
	t.Setenv(EnvClickhouseDeleteOnCluster, "true")

	q := NewClickhouseReplicated()

	assertSQL(t, "CreateTable", q.CreateTable(testTable), `
		CREATE TABLE IF NOT EXISTS goose_db_version ON CLUSTER envcluster (
			version_id Int64,
			is_applied UInt8,
			tstamp DateTime64(6) DEFAULT now64(6)
		)
		ENGINE = ReplicatedMergeTree('/zk/env', 'env-replica')
		ORDER BY (version_id)`)

	assertSQL(t, "InsertVersion", q.InsertVersion(testTable),
		`INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, $2) SETTINGS insert_quorum=2, select_sequential_consistency=1`)

	assertSQL(t, "DeleteVersion", q.DeleteVersion(testTable),
		`ALTER TABLE goose_db_version ON CLUSTER envcluster DELETE WHERE version_id = $1 SETTINGS mutations_sync = 0`)
}

func TestClickhouseReplicated_OptionsOverrideEnv(t *testing.T) {
	t.Setenv(EnvClickhouseCluster, "envcluster")
	t.Setenv(EnvClickhouseMutationsSync, "5")

	q := NewClickhouseReplicated(
		WithClickhouseCluster("optcluster"),
		WithClickhouseMutationsSync(1),
	)

	if got := q.CreateTable(testTable); !strings.Contains(got, "ON CLUSTER optcluster") {
		t.Errorf("expected option to override env cluster; got: %s", got)
	}
	if got := q.DeleteVersion(testTable); !strings.Contains(got, "mutations_sync = 1") {
		t.Errorf("expected option to override env mutations_sync; got: %s", got)
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
		got := q.InsertVersion(testTable)
		if !strings.Contains(got, "insert_quorum="+tc.want+",") {
			t.Errorf("quorum %q: expected insert_quorum=%s in SQL, got: %s", tc.in, tc.want, got)
		}
	}
}
