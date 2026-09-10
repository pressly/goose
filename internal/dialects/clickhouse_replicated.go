package dialects

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/pressly/goose/v3/database/dialect"
)

// Environment variable names used to configure the clickhouse-replicated dialect.
const (
	EnvClickhouseCluster      = "GOOSE_CLICKHOUSE_CLUSTER"
	EnvClickhouseZKPath       = "GOOSE_CLICKHOUSE_ZK_PATH"
	EnvClickhouseReplicaName  = "GOOSE_CLICKHOUSE_REPLICA_NAME"
	EnvClickhouseInsertQuorum = "GOOSE_CLICKHOUSE_INSERT_QUORUM"
)

// ClickhouseReplicatedOption configures the clickhouse-replicated dialect.
type ClickhouseReplicatedOption func(*clickhouseReplicated)

// WithClickhouseCluster sets the ClickHouse cluster name used in ON CLUSTER
// clauses. Overrides GOOSE_CLICKHOUSE_CLUSTER. This value is required.
func WithClickhouseCluster(v string) ClickhouseReplicatedOption {
	return func(c *clickhouseReplicated) { c.cluster = v }
}

// WithClickhouseZooKeeperPath sets an explicit ZooKeeper/Keeper path for the
// ReplicatedMergeTree engine. Overrides GOOSE_CLICKHOUSE_ZK_PATH. When empty
// (the default) the server macros are used instead.
func WithClickhouseZooKeeperPath(v string) ClickhouseReplicatedOption {
	return func(c *clickhouseReplicated) { c.zkPath = v }
}

// WithClickhouseReplicaName sets an explicit replica name for the
// ReplicatedMergeTree engine. Overrides GOOSE_CLICKHOUSE_REPLICA_NAME. When
// empty (the default) the server macros are used instead.
func WithClickhouseReplicaName(v string) ClickhouseReplicatedOption {
	return func(c *clickhouseReplicated) { c.replicaName = v }
}

// WithClickhouseInsertQuorum sets the insert_quorum setting used on both the
// InsertVersion (up-migration) and DeleteVersion (down-migration tombstone)
// queries. Overrides GOOSE_CLICKHOUSE_INSERT_QUORUM. Default: "auto".
func WithClickhouseInsertQuorum(v string) ClickhouseReplicatedOption {
	return func(c *clickhouseReplicated) { c.insertQuorum = v }
}

// NewClickhouseReplicated returns a new [dialect.Querier] for the
// clickhouse-replicated dialect. Configuration is read from GOOSE_CLICKHOUSE_*
// environment variables and can be overridden per option.
//
// The dialect uses an insert-mostly design: down-migrations insert a
// tombstone row (is_applied = 0) instead of issuing an ALTER ... DELETE
// mutation, per ClickHouse best practice
// (https://clickhouse.com/docs/concepts/best-practices/avoid-mutations).
// Duplicate rows for the same version_id are collapsed automatically by
// background merges of the ReplicatedReplacingMergeTree engine, and read
// queries derive current state per version using argMax keyed on (tstamp,
// tombstone-wins-on-tie), so the latest row wins, and a tombstone wins any
// tie against an apply row with the same tstamp.
// Reads also carry select_sequential_consistency=1 so that, when writes were
// quorum-committed (see WithClickhouseInsertQuorum), a read landing on a
// lagging replica will wait for it to catch up to the last quorum insert to
// the version table. That only scopes visibility of already-written
// bookkeeping rows; it is not a compare-and-swap on the insert, does not
// serialize concurrent readers, and does not coordinate migration DDL. This
// dialect therefore does not make it safe to run multiple goose migrators
// concurrently against the same cluster — external interlocking is still
// required.
//
// GOOSE_CLICKHOUSE_CLUSTER (or [WithClickhouseCluster]) is required; if empty,
// [ErrClickhouseReplicatedNoCluster] is returned.
func NewClickhouseReplicated(opts ...ClickhouseReplicatedOption) (dialect.Querier, error) {
	c := &clickhouseReplicated{
		cluster:      os.Getenv(EnvClickhouseCluster),
		zkPath:       os.Getenv(EnvClickhouseZKPath),
		replicaName:  os.Getenv(EnvClickhouseReplicaName),
		insertQuorum: os.Getenv(EnvClickhouseInsertQuorum),
	}
	if c.insertQuorum == "" {
		c.insertQuorum = "auto"
	}
	for _, opt := range opts {
		opt(c)
	}
	if c.cluster == "" {
		return nil, ErrClickhouseReplicatedNoCluster
	}
	return c, nil
}

type clickhouseReplicated struct {
	cluster      string
	zkPath       string
	replicaName  string
	insertQuorum string
}

var _ dialect.Querier = (*clickhouseReplicated)(nil)

// ErrClickhouseReplicatedNoCluster is returned by [NewClickhouseReplicated]
// if no cluster name is configured.
var ErrClickhouseReplicatedNoCluster = errors.New(
	"clickhouse-replicated: cluster name is required; set " +
		EnvClickhouseCluster + " or pass WithClickhouseCluster(...)",
)

// quorumSetting returns the insert_quorum value formatted for inclusion in a
// SETTINGS clause. Numeric values are emitted bare; symbolic values ("auto",
// "off") are single-quoted.
func (c *clickhouseReplicated) quorumSetting() string {
	if _, err := strconv.Atoi(c.insertQuorum); err == nil {
		return c.insertQuorum
	}
	return "'" + c.insertQuorum + "'"
}

func (c *clickhouseReplicated) CreateTable(tableName string) string {
	engine := "ReplicatedReplacingMergeTree(tstamp)"
	if c.zkPath != "" && c.replicaName != "" {
		engine = fmt.Sprintf("ReplicatedReplacingMergeTree('%s', '%s', tstamp)", c.zkPath, c.replicaName)
	}
	q := `CREATE TABLE IF NOT EXISTS %s ON CLUSTER %s (
		version_id Int64,
		is_applied UInt8,
		tstamp DateTime64(6) DEFAULT now64(6)
	)
	ENGINE = %s
	ORDER BY (version_id)`
	return fmt.Sprintf(q, tableName, c.cluster, engine)
}

func (c *clickhouseReplicated) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) SETTINGS insert_quorum=%s VALUES ($1, $2)`
	return fmt.Sprintf(q, tableName, c.quorumSetting())
}

// DeleteVersion records a down-migration by inserting a tombstone row
// (is_applied = 0) with a fresh tstamp. Read queries collapse duplicate rows
// per version_id using [tombstoneWinsExpr] so the tombstone wins over any
// earlier is_applied = 1 row, even one with an identical tstamp. Background
// merges of the
// ReplicatedReplacingMergeTree engine eventually physically collapse the
// duplicates. This avoids ALTER ... DELETE mutations per ClickHouse best
// practice (https://clickhouse.com/docs/concepts/best-practices/avoid-mutations).
func (c *clickhouseReplicated) DeleteVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) SETTINGS insert_quorum=%s VALUES ($1, 0)`
	return fmt.Sprintf(q, tableName, c.quorumSetting())
}

// tombstoneWinsExpr is the argMax comparison key used to collapse duplicate
// rows per version_id: order by tstamp, and on a tie prefer the tombstone
// (is_applied = 0) over the apply row. Ties are only possible if two writes
// for the same version_id land on the same microsecond-resolution tstamp,
// which requires concurrent, uninterlocked writers — a usage pattern this
// dialect already documents as unsupported — but breaking ties explicitly
// removes the need to rely on that assumption for correctness.
const tombstoneWinsExpr = "tuple(tstamp, is_applied = 0)"

// GetMigrationByVersion must return no row for a version whose latest state
// is a tombstone (is_applied = 0): [database.Store.GetMigration] and its
// callers (Provider.status, Provider.apply) treat "row found" as "currently
// applied" and never inspect the returned is_applied column, matching the
// other dialects' delete-based semantics where a row only exists while
// applied. The outer is_applied = 1 filter, mirroring [ListMigrations],
// preserves that contract for this dialect's insert-only tombstone design.
func (c *clickhouseReplicated) GetMigrationByVersion(tableName string) string {
	// The inner max(tstamp) is aliased to something other than "tstamp":
	// aliasing it back to "tstamp" makes ClickHouse substitute that alias
	// into the argMax(...) tuple expression on the same SELECT line (which
	// still needs the raw per-row tstamp column), nesting one aggregate
	// inside another and erroring with "Aggregate function ... is found
	// inside another aggregate function".
	q := `SELECT ts AS tstamp, is_applied FROM (
	SELECT version_id, max(tstamp) AS ts, argMax(is_applied, %[2]s) AS is_applied FROM %[1]s WHERE version_id = $1 GROUP BY version_id
) WHERE is_applied = 1 SETTINGS select_sequential_consistency=1`
	return fmt.Sprintf(q, tableName, tombstoneWinsExpr)
}

func (c *clickhouseReplicated) ListMigrations(tableName string) string {
	// Only surface currently-applied versions. Because this dialect records
	// down-migrations as tombstone inserts (is_applied = 0) rather than by
	// ALTER ... DELETE, tombstoned rows still exist and would otherwise be
	// reported to goose as "in DB" — which the provider's UpVersions logic
	// then treats as "already applied", suppressing re-application.
	q := `SELECT version_id, is_applied FROM (
	SELECT version_id, argMax(is_applied, %[2]s) AS is_applied FROM %[1]s GROUP BY version_id
) WHERE is_applied = 1 ORDER BY version_id DESC SETTINGS select_sequential_consistency=1`
	return fmt.Sprintf(q, tableName, tombstoneWinsExpr)
}

func (c *clickhouseReplicated) GetLatestVersion(tableName string) string {
	q := `SELECT max(version_id) FROM (SELECT version_id, argMax(is_applied, %[2]s) AS is_applied FROM %[1]s GROUP BY version_id) WHERE is_applied = 1 SETTINGS select_sequential_consistency=1`
	return fmt.Sprintf(q, tableName, tombstoneWinsExpr)
}
