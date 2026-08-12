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
	EnvClickhouseCluster         = "GOOSE_CLICKHOUSE_CLUSTER"
	EnvClickhouseZKPath          = "GOOSE_CLICKHOUSE_ZK_PATH"
	EnvClickhouseReplicaName     = "GOOSE_CLICKHOUSE_REPLICA_NAME"
	EnvClickhouseInsertQuorum    = "GOOSE_CLICKHOUSE_INSERT_QUORUM"
	EnvClickhouseMutationsSync   = "GOOSE_CLICKHOUSE_MUTATIONS_SYNC"
	EnvClickhouseDeleteOnCluster = "GOOSE_CLICKHOUSE_DELETE_ON_CLUSTER"
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

// WithClickhouseInsertQuorum sets the insert_quorum setting used on the
// InsertVersion query. Overrides GOOSE_CLICKHOUSE_INSERT_QUORUM. Default:
// "auto".
func WithClickhouseInsertQuorum(v string) ClickhouseReplicatedOption {
	return func(c *clickhouseReplicated) { c.insertQuorum = v }
}

// WithClickhouseMutationsSync sets the mutations_sync setting used on the
// DeleteVersion query. Overrides GOOSE_CLICKHOUSE_MUTATIONS_SYNC. Default: 2.
func WithClickhouseMutationsSync(n int) ClickhouseReplicatedOption {
	return func(c *clickhouseReplicated) { c.mutationsSync = n }
}

// WithClickhouseDeleteOnCluster controls whether the DeleteVersion query
// includes an ON CLUSTER clause. Overrides GOOSE_CLICKHOUSE_DELETE_ON_CLUSTER.
// Default: false (rely on ReplicatedMergeTree replication log).
func WithClickhouseDeleteOnCluster(v bool) ClickhouseReplicatedOption {
	return func(c *clickhouseReplicated) { c.deleteOnCluster = v }
}

// NewClickhouseReplicated returns a new [dialect.Querier] for the
// clickhouse-replicated dialect. Configuration is read from GOOSE_CLICKHOUSE_*
// environment variables and can be overridden per option.
//
// GOOSE_CLICKHOUSE_CLUSTER (or [WithClickhouseCluster]) is required; if empty,
// [CreateTable] returns an error explaining how to set it.
func NewClickhouseReplicated(opts ...ClickhouseReplicatedOption) dialect.Querier {
	c := &clickhouseReplicated{
		cluster:         os.Getenv(EnvClickhouseCluster),
		zkPath:          os.Getenv(EnvClickhouseZKPath),
		replicaName:     os.Getenv(EnvClickhouseReplicaName),
		insertQuorum:    os.Getenv(EnvClickhouseInsertQuorum),
		mutationsSync:   2,
		deleteOnCluster: false,
	}
	if c.insertQuorum == "" {
		c.insertQuorum = "auto"
	}
	if v := os.Getenv(EnvClickhouseMutationsSync); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.mutationsSync = n
		}
	}
	if v := os.Getenv(EnvClickhouseDeleteOnCluster); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			c.deleteOnCluster = b
		}
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

type clickhouseReplicated struct {
	cluster         string
	zkPath          string
	replicaName     string
	insertQuorum    string
	mutationsSync   int
	deleteOnCluster bool
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

func (c *clickhouseReplicated) CreateTable(tableName string) string {
	if c.cluster == "" {
		panic(ErrClickhouseReplicatedNoCluster)
	}
	engine := "ReplicatedMergeTree"
	if c.zkPath != "" && c.replicaName != "" {
		engine = fmt.Sprintf("ReplicatedMergeTree('%s', '%s')", c.zkPath, c.replicaName)
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
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2) SETTINGS insert_quorum=%s, select_sequential_consistency=1`
	quorum := c.insertQuorum
	if _, err := strconv.Atoi(quorum); err != nil {
		// non-numeric quorum values ("auto", "off") must be quoted
		quorum = "'" + quorum + "'"
	}
	return fmt.Sprintf(q, tableName, quorum)
}

func (c *clickhouseReplicated) DeleteVersion(tableName string) string {
	if c.deleteOnCluster {
		q := `ALTER TABLE %s ON CLUSTER %s DELETE WHERE version_id = $1 SETTINGS mutations_sync = %d`
		return fmt.Sprintf(q, tableName, c.cluster, c.mutationsSync)
	}
	q := `ALTER TABLE %s DELETE WHERE version_id = $1 SETTINGS mutations_sync = %d`
	return fmt.Sprintf(q, tableName, c.mutationsSync)
}

func (c *clickhouseReplicated) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouseReplicated) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouseReplicated) GetLatestVersion(tableName string) string {
	q := `SELECT max(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}
