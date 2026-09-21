package dialects

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// normalizeSQL collapses whitespace so tests can compare SQL strings without
// being sensitive to formatting.
func normalizeSQL(s string) string {
	return strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(s, " "))
}

func assertSQL(t *testing.T, method, got, want string) {
	t.Helper()
	assert.Equal(t, normalizeSQL(want), normalizeSQL(got), "%s SQL mismatch", method)
}

const testTable = "goose_db_version"

func TestClickhouseReplicated_AllOptionsSet(t *testing.T) {
	q, err := NewClickhouseReplicated(
		WithClickhouseCluster("goose_cluster"),
		WithClickhouseZooKeeperPath("/clickhouse/tables/{shard}/goose_db_version"),
		WithClickhouseReplicaName("{replica}"),
		WithClickhouseInsertQuorum("3"),
	)
	require.NoError(t, err)

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
		`SELECT ts AS tstamp, is_applied FROM (
	SELECT version_id, max(tstamp) AS ts, argMax(is_applied, tuple(tstamp, is_applied = 0)) AS is_applied FROM goose_db_version WHERE version_id = $1 GROUP BY version_id
) WHERE is_applied = 1 SETTINGS select_sequential_consistency=1`)

	assertSQL(t, "ListMigrations", q.ListMigrations(testTable),
		`SELECT version_id, is_applied FROM (
	SELECT version_id, argMax(is_applied, tuple(tstamp, is_applied = 0)) AS is_applied FROM goose_db_version GROUP BY version_id
) WHERE is_applied = 1 ORDER BY version_id DESC SETTINGS select_sequential_consistency=1`)

	assertSQL(t, "GetLatestVersion", q.GetLatestVersion(testTable),
		`SELECT max(version_id) FROM (SELECT version_id, argMax(is_applied, tuple(tstamp, is_applied = 0)) AS is_applied FROM goose_db_version GROUP BY version_id) WHERE is_applied = 1 SETTINGS select_sequential_consistency=1`)
}

// TestClickhouseReplicated_TableExists_EngineGuard verifies the TableExists query embeds a
// throwIf guard that raises a clear error if the table exists with an engine other than
// ReplicatedReplacingMergeTree (i.e. it was created by the stock clickhouse dialect instead).
func TestClickhouseReplicated_TableExists_EngineGuard(t *testing.T) {
	q, err := NewClickhouseReplicated(WithClickhouseCluster("c"))
	require.NoError(t, err)
	querier, ok := q.(interface{ TableExists(string) string })
	require.True(t, ok, "querier does not implement TableExists")
	got := querier.TableExists(testTable)
	for _, want := range []string{
		"system.tables",
		"throwIf(",
		"ReplicatedReplacingMergeTree",
		"the stock clickhouse dialect",
		testTable,
	} {
		assert.Contains(t, got, want, "TableExists SQL missing %q", want)
	}
}

func TestClickhouseReplicated_EnvDefaults(t *testing.T) {
	// Only the required env var; everything else should hit defaults.
	t.Setenv(EnvClickhouseCluster, "envcluster")
	t.Setenv(EnvClickhouseZKPath, "")
	t.Setenv(EnvClickhouseReplicaName, "")
	t.Setenv(EnvClickhouseInsertQuorum, "")

	q, err := NewClickhouseReplicated()
	require.NoError(t, err)

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

	q, err := NewClickhouseReplicated()
	require.NoError(t, err)

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

	q, err := NewClickhouseReplicated(
		WithClickhouseCluster("optcluster"),
		WithClickhouseInsertQuorum("1"),
	)
	require.NoError(t, err)

	assert.Contains(t, q.CreateTable(testTable), "ON CLUSTER optcluster", "expected option to override env cluster")
	assert.Contains(t, q.InsertVersion(testTable), "insert_quorum=1", "expected option to override env insert_quorum on InsertVersion")
	assert.Contains(t, q.DeleteVersion(testTable), "insert_quorum=1", "expected option to override env insert_quorum on DeleteVersion")
}

func TestClickhouseReplicated_MissingClusterErrors(t *testing.T) {
	t.Setenv(EnvClickhouseCluster, "")

	q, err := NewClickhouseReplicated()
	require.Nil(t, q, "expected nil querier on error")
	require.ErrorIs(t, err, ErrClickhouseReplicatedNoCluster)

	msg := err.Error()
	assert.Contains(t, msg, EnvClickhouseCluster, "expected error to mention env var")
	assert.Contains(t, msg, "WithClickhouseCluster", "expected error to mention WithClickhouseCluster")
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
		q, err := NewClickhouseReplicated(
			WithClickhouseCluster("c"),
			WithClickhouseInsertQuorum(tc.in),
		)
		require.NoError(t, err)
		for _, sql := range []struct {
			name, got string
		}{
			{"InsertVersion", q.InsertVersion(testTable)},
			{"DeleteVersion", q.DeleteVersion(testTable)},
		} {
			assert.Contains(t, sql.got, "insert_quorum="+tc.want, "%s quorum %q", sql.name, tc.in)
		}
	}
}
