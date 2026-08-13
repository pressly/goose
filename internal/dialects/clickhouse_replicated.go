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
// queries derive current state per version using argMax(is_applied, tstamp)
// with select_sequential_consistency=1 for cross-replica read-after-write
// correctness.
//
// GOOSE_CLICKHOUSE_CLUSTER (or [WithClickhouseCluster]) is required; if empty,
// [CreateTable] returns an error explaining how to set it.
func NewClickhouseReplicated(opts ...ClickhouseReplicatedOption) dialect.Querier {
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
	return c
}

type clickhouseReplicated struct {
	cluster      string
	zkPath       string
	replicaName  string
	insertQuorum string
}

var _ dialect.Querier = (*clickhouseReplicated)(nil)

// ErrClickhouseReplicatedNoCluster is returned by validation entry points
// (e.g. [database.NewStore] when constructing the clickhouse-replicated
// dialect) if no cluster name is configured. The Querier's own [CreateTable]
// panics with the same error as a defensive check, but callers should
// validate up front so the misconfiguration is caught before any SQL is
// executed.
var ErrClickhouseReplicatedNoCluster = errors.New(
	"clickhouse-replicated: cluster name is required; set " +
		EnvClickhouseCluster + " or pass WithClickhouseCluster(...)",
)

// Cluster reports the configured cluster name. It is used by callers that
// wish to validate the querier before executing any SQL.
func (c *clickhouseReplicated) Cluster() string { return c.cluster }

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
	if c.cluster == "" {
		panic(ErrClickhouseReplicatedNoCluster)
	}
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
// per version_id using argMax(is_applied, tstamp) so the tombstone wins over
// any earlier is_applied = 1 row. Background merges of the
// ReplicatedReplacingMergeTree engine eventually physically collapse the
// duplicates. This avoids ALTER ... DELETE mutations per ClickHouse best
// practice (https://clickhouse.com/docs/concepts/best-practices/avoid-mutations).
func (c *clickhouseReplicated) DeleteVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) SETTINGS insert_quorum=%s VALUES ($1, 0)`
	return fmt.Sprintf(q, tableName, c.quorumSetting())
}

func (c *clickhouseReplicated) GetMigrationByVersion(tableName string) string {
	q := `SELECT argMax(tstamp, tstamp) AS tstamp, argMax(is_applied, tstamp) AS is_applied FROM %s WHERE version_id = $1 GROUP BY version_id SETTINGS select_sequential_consistency=1`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouseReplicated) ListMigrations(tableName string) string {
	// Only surface currently-applied versions. Because this dialect records
	// down-migrations as tombstone inserts (is_applied = 0) rather than by
	// ALTER ... DELETE, tombstoned rows still exist and would otherwise be
	// reported to goose as "in DB" — which the provider's UpVersions logic
	// then treats as "already applied", suppressing re-application.
	q := `SELECT version_id, is_applied FROM (
	SELECT version_id, argMax(is_applied, tstamp) AS is_applied FROM %s GROUP BY version_id
) WHERE is_applied = 1 ORDER BY version_id DESC SETTINGS select_sequential_consistency=1`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouseReplicated) GetLatestVersion(tableName string) string {
	q := `SELECT max(version_id) FROM (SELECT version_id, argMax(is_applied, tstamp) AS is_applied FROM %s GROUP BY version_id) WHERE is_applied = 1 SETTINGS select_sequential_consistency=1`
	return fmt.Sprintf(q, tableName)
}
