package database

import (
	"github.com/pressly/goose/v3/database/dialect"
	"github.com/pressly/goose/v3/internal/dialects"
)

// ClickhouseReplicatedOption configures the clickhouse-replicated dialect. See
// [NewClickhouseReplicated] for details and the list of With* option
// constructors.
type ClickhouseReplicatedOption = dialects.ClickhouseReplicatedOption

// Environment variable names read by the clickhouse-replicated dialect.
const (
	EnvClickhouseCluster      = dialects.EnvClickhouseCluster
	EnvClickhouseZKPath       = dialects.EnvClickhouseZKPath
	EnvClickhouseReplicaName  = dialects.EnvClickhouseReplicaName
	EnvClickhouseInsertQuorum = dialects.EnvClickhouseInsertQuorum
)

// ErrClickhouseReplicatedNoCluster is returned by [NewClickhouseReplicated]
// and [NewStore] (for [DialectClickHouseReplicated]) when the cluster name is
// not configured via either [WithClickhouseCluster] or GOOSE_CLICKHOUSE_CLUSTER.
var ErrClickhouseReplicatedNoCluster = dialects.ErrClickhouseReplicatedNoCluster

// WithClickhouseCluster sets the ClickHouse cluster name used in ON CLUSTER
// clauses. Overrides GOOSE_CLICKHOUSE_CLUSTER. This value is required.
func WithClickhouseCluster(v string) ClickhouseReplicatedOption {
	return dialects.WithClickhouseCluster(v)
}

// WithClickhouseZooKeeperPath sets an explicit ZooKeeper/Keeper path for the
// ReplicatedMergeTree engine. Overrides GOOSE_CLICKHOUSE_ZK_PATH. When empty
// (the default) the server macros are used instead.
func WithClickhouseZooKeeperPath(v string) ClickhouseReplicatedOption {
	return dialects.WithClickhouseZooKeeperPath(v)
}

// WithClickhouseReplicaName sets an explicit replica name for the
// ReplicatedMergeTree engine. Overrides GOOSE_CLICKHOUSE_REPLICA_NAME. When
// empty (the default) the server macros are used instead.
func WithClickhouseReplicaName(v string) ClickhouseReplicatedOption {
	return dialects.WithClickhouseReplicaName(v)
}

// WithClickhouseInsertQuorum sets the insert_quorum setting used on both the
// InsertVersion (up-migration) and DeleteVersion (down-migration tombstone)
// queries. Overrides GOOSE_CLICKHOUSE_INSERT_QUORUM. Default: "auto".
func WithClickhouseInsertQuorum(v string) ClickhouseReplicatedOption {
	return dialects.WithClickhouseInsertQuorum(v)
}

// NewClickhouseReplicated returns a [dialect.Querier] for the
// clickhouse-replicated dialect. Configuration is read from
// GOOSE_CLICKHOUSE_* environment variables and can be overridden per option.
//
// The returned querier is validated up front: if no cluster name is
// configured, [ErrClickhouseReplicatedNoCluster] is returned.
//
// The result is intended to be passed to [NewStoreFromQuerier]. Library
// callers that only need env-var configuration can equivalently call
// [NewStore] with [DialectClickHouseReplicated].
func NewClickhouseReplicated(opts ...ClickhouseReplicatedOption) (dialect.Querier, error) {
	q := dialects.NewClickhouseReplicated(opts...)
	if v, ok := q.(interface{ Cluster() string }); ok && v.Cluster() == "" {
		return nil, ErrClickhouseReplicatedNoCluster
	}
	return q, nil
}
