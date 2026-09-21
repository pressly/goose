# ClickHouse Dialects

Goose has two dialects for ClickHouse - the `clickhouse` dialect for standalone
ClickHouse, and the `clickhouse-replicated` dialect that adds support for
clustered ClickHouse databases.

It is safe to use `clickhouse-replicated` for standalone ClickHouse installs.

    XXX TODO what about clickhouse cloud?
    XXX When (if) to still use the original clickhouse dialect?

## ClickHouse replicated cluster dialect

`clickhouse-replicated` is a replacement for the `clickhouse` dialect that
supports multi-replica ClickHouse clusters by using `ON CLUSTER` for the
metadata table and using safer strategies for updating its rows.

> [!IMPORTANT]
>
> This dialect does **not** make it safe to run `goose` migrators concurrently
> against the same cluster. See [limitations](#limitations).

Users select the dialect explicitly.

The column layout deliberately differs from stock `clickhouse` so an old CLI
cannot silently mis-mutate a replicated table. Attempting to run the
`clickhouse` dialect on a database managed by `clickhouse-replicated` (or vice
versa) will fail with an error.

There is no automated migration from the stock `clickhouse` dialect to
`clickhouse-replicated`. To migrate it is necessary to manually drop the old
metadata table and manually create a new one.

    XXX TODO add a conversion flag, because that's a mess?

### Configuration

Configuration comes from `GOOSE_CLICKHOUSE_*` environment variables when run
via the CLI, or `database.NewClickhouseReplicated(...)` options when embedded
into a golang program:

| Env var                              | Option                             | Default | Notes                                           |
|--------------------------------------|------------------------------------|---------|-------------------------------------------------|
| `GOOSE_CLICKHOUSE_CLUSTER`           | `WithClickhouseCluster`            | *(none)*| **Required.** Cluster name for `ON CLUSTER` DDL.|
| `GOOSE_CLICKHOUSE_ZK_PATH`           | `WithClickhouseZooKeeperPath`      | *(empty)* | Empty → rely on `default_replica_path` macro. |
| `GOOSE_CLICKHOUSE_REPLICA_NAME`      | `WithClickhouseReplicaName`        | *(empty)* | Empty → rely on `default_replica_name` macro. |
| `GOOSE_CLICKHOUSE_INSERT_QUORUM`     | `WithClickhouseInsertQuorum`       | `auto`  | Numeric values (e.g. `3`) or `auto`/`off`. Applied to both up and down writes. |

### Limitations

#### Not safe for concurrent execution on different nodes

    XXX TODO how to wait for ZK flush, ensure consistency?
    XXX TODO down/partitioned nodes safety

See [multi-node safety](#multi-node-safety) for implementation details.



## Implementation details

### Waiting for DDL

    XXX TODO needs test cover for down-CH majority and minority cases

### Avoids mutations

The dialect follows an **insert-mostly** design in line with ClickHouse's
[avoid mutations](https://clickhouse.com/docs/concepts/best-practices/avoid-mutations)
guidance and try to maximise cluster-consistency for DDL:

- Up-migrations `INSERT` a row with `is_applied = 1`.
- Down-migrations `INSERT` a tombstone row with `is_applied = 0` — **no `ALTER
  … DELETE` mutation is ever issued.**
- Both up and down writes carry an `insert_quorum` setting.

  XXX TODO What about the DDL op itself, and quorum for that? If ZK has
  quorum it can write the DDL, even though CH may not then actually apply
  it.

- Read queries derive the current state per `version_id` using `argMax` keyed
  on `(tstamp, is_applied = 0)`, so the latest-`tstamp` row wins, and a
  tombstone wins any tie against an apply row sharing the same `tstamp`. All
  reads set `select_sequential_consistency=1`, which — for writes that actually
  went through a quorum — makes a query landing on a lagging replica wait until
  it has caught up to the last quorum-committed write to `goose_db_version`.
  This only affects visibility of already-written bookkeeping rows on the read
  side; it does not serialize concurrent readers against each other or provide
  any coordination for the migration DDL itself. With `insert_quorum='off'` or
  `'1'` even that read-side guard is a no-op.
- Duplicate rows for the same `version_id` are collapsed automatically by
  background merges of the `ReplacingMergeTree` engine; no manual pruning is
  required.

### Multi-node safety

This dialect cannot make it safe to run `goose` concurrently against multiple
nodes. ClickHouse does not offer a distributed-lock primitive suitable for
cross-cluster mutual exclusion, so concurrent-run interlocking must be arranged
outside of ClickHouse. Clickhouse lacks distributed locking such as advisory
locks, user-level explicit table locks, or non-autocommit DDL that could be used
for this purpose.

For non-cloud hosted ClickHouse where direct access to Keeper is possible,
mutual exclusion could be enforced by embedding Goose in a supervisor program
that uses a direct Keeper connection to ensure mutual exclusion. Clickhouse
Cloud does not expose Keeper directly to end users, so this option is not
available there. It isn't implemented in this dialect to avoid the need to
carry a Keeper client and additional complexity in Goose itself.

The `insert_quorum` and `select_sequential_consistency=1` options only scope
the visibility of individual `goose_db_version` rows. They do not serialize
concurrent readers against each other, do not turn the bookkeeping insert into
a compare-and-swap, and give no coordination for the migration DDL itself. Two
racing goose runs can each read the same "latest applied" state, each decide
the next migration is theirs to run, and each submit its `ON CLUSTER` DDL to
the Keeper DDL queue independently.

Even if the DDL itself is safe to run concurrently (such as two identical
`CREATE TABLE ... IF NOT EXISTS ... ON CLUSTER` statements) the inability to
bundle a DDL statement in the same transaction as Goose's metadata update
means it's unsafe to 

### Testing

The `TestClickhouseReplicated` integration test brings up a two-node cluster
(see [`internal/testing/integration/clickhouse-replicated/`](./internal/testing/integration/clickhouse-replicated/README.md))
via `ory/dockertest` and is exercised by the standard `test-integration` CI job.
