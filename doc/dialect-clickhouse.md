# ClickHouse Dialect

## Usage

Use Goose on _standalone ClickHouse_ databases, on ClickHouse Cloud databases
(with [caveats below](#limitations)), or _per-node_ on clustered ClickHouse
with the `Atomic` database engine.

> [!WARNING]
> The current Goose `clickhouse` migrator is _not_ well suited for use on
> clustered `ClickHouse` with the `Replicated` engine, or on the `Atomic`
> engine using `WITH CLUSTER` DDL. See "Limitations" below. Such use is
> possible, but requires additional setup and careful workarounds.

All migrations must be annotated with `-- +goose NO TRANSACTION` as the first
line in the file, e.g.

```sql
-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE foo (...) IF NOT EXISTS;

-- +goose Down
DROP TABLE foo IF EXISTS;
```

> [!WARNING]
>
> Goose-managed DDL operations should be _idempotent_ - i.e., running the same
> DDL operation again (with no other DDL operation in between) will have no
> effect.
>
> This is because goose may re-run any given migration multiple times before it
> proceeds to the next migration. ClickHouse lacks support for committing a
> data change along with a DDL operation in a single transaction, so Goose
> cannot atomically update its metadata along with running the migration DDL
> itself. See [Limitations](#limitations).

## Limitations

### Requires idempotent DDL

Because ClickHouse itself lacks support for running a DDL operation in an
atomic transaction alongside a goose metadata table update, Goose cannot
guarantee that any given migration will run exactly once.

If Goose or Clickhouse crashes (or the connection is interrupted) after a DDL
is issued but before Goose sends the metadata update to ClickHouse, Goose
will not know that the migration ran and will re-try it from the beginning.
Even for single-statement DDL.

Additionally, [ClickHouse has no form of DDL tagging or server-side DDL
idempotency](https://github.com/clickhouse/clickhouse/issues/95963).
If Goose sends a DDL statement then loses its connection before
ClickHouse confirms receipt of the DDL, Goose has no way to know whether
ClickHouse received the DDL and began executing it or not. It therefore has no
way to know whether it needs to be re-run or not, especially if Goose connects
to a different node on restart. ClickHouse has no cluster-wide view of
in-progress and completed DDL that Goose could inspect, and any such check
would be subject to race conditions in any case.

**It is therefore only safe to use Goose to run DDL that can be harmlessly run
repeatedly.**

To run non-idempotent DDL, a unique golang migration must be implemented for
that specific DDL operation. This must introspect the database state to check
for and wait for a matching currently-running DDL after waiting a safety-period
to allow for queries in-flight on the wire. Once any running DDL has completed,
it must inspect the database catalog state to determine whether or not the DDL
was already run. If it already ran, it must return a no-op success; otherwise
it must re-run the DDL. Goose cannot implement this in a generic manner, as there
is no way to ask ClickHouse "did this DDL already run" for any given generic
DDL statement.

### Asynchronous migrations

By default, ClickHouse DDL execution returns success to the client when the
_metadata changes_ are complete. Some DDL runs asynchronous mutations of data
part-files. This mutation begins only once the DDL commits, potentially after a
delay. ClickHouse does not report to the client whether or not a given DDL
execution triggered a mutation. By default ClickHouse does not wait for the
mutation to complete before returning success to the client - this is
controlled by `mutations_sync`.

A migration _can_ `SET mutations_sync=2` before running a mutating migration to
wait for all nodes to apply the mutation; _however_ this greatly increases the
risk of the migration step timing out. In that case, as ClickHouse never
replied to tell Goose that the DDL itself committed  Goose will not have
updated its metadata table. So Goose will re-run the DDL that triggered the
migration when it is re-run, likely timing out again.

Accordingly it's generally better to run with `mutations_sync=0`. The mutation
will be queued and run in the background. Subsequent queries mutating the same
table will block if a mutation is still in progress on the table. This can
cause timeouts on a subsequent mutation step, but a retry loop around Goose
will handle that.

### Limitations with clustered ClickHouse

Goose's `clickhouse` dialect has several limitations affecting use on clustered
ClickHouse databases:

* It currently creates its version table with a non-replicated `MergeTree`
  table engine;
* It does not enforce cluster-wide read consistency when reading its version
  table; and
* The asynchronous `DELETE`s used for down-migrations are slow and do not
  support read consistency.

Whether Goose is safe to use against a given ClickHouse deployment depends on
whether ClickHouse itself rewrites that plain `MergeTree()` engine into
something that actually replicates row data, which varies by deployment.

The only ClickHouse deployment type that does not require any
deployment-type-specific workarounds is a standalone (non-clustered) ClickHouse
(generally using the `Atomic` or `Ordinary` database engines).

Workarounds and customisation are required for all other ClickHouse deployment
models:

#### Clickhouse Cloud

Cloud transparently substitutes `SharedMergeTree` (and the matching `Shared*`
variant for other MergeTree-family engines) for whatever engine a `CREATE
TABLE` requests with no changes needed to the DDL. This ensures that Goose's
metadata is replicated across the cluster.

To prevent races on `up`, **the clickhouse user or role that Goose connects as
should have a settings profile that enforces
`select_sequential_consistency=1`** - or care must be taken to allow time for
all replicas to catch up before running goose again. For applications embedding
Goose, an alternative is
[a custom Store that uses `SYSTEM SYNC REPLICA ... LIGHTWEIGHT`](#appendix-system-sync-replica-via-a-custom-store).

Goose down-migrations use `ALTER ... DELETE` mutations. While Goose uses
`mutations_sync = 2` for these, that only makes that Goose instance wait for
completion. It doesn't ensure that other Goose executions (such as a retry
after Goose crashes/disconnects while updating down-migration metadata) will
actually see the new metadata. `ALTER ... DELETE` is asynchronous so even on
the same node, a retry might see the old Goose from before the down-migration,
despite the fact that the actual down-migration DDL has already been completed.
So all replicas must still be allowed time to finish applying a down-migration,
and to be confirmed as having done so, before another Goose invocation is
started. Consider monitoring `system.mutations`.

Externally managed mutual excusion to prevent concurrent Goose migrator runs is
strongly recommended.

#### Clickhouse clustered with `Atomic` database engine

It is safe to run goose on a clustered clickhouse that uses the atomic
database engine _only_ by:

* treating each node as an individual ClickHouse database with its own per-node
  migration state, _never_ using `ON CLUSTER` for any DDL operation; or
* selecting a single "leader" ClickHouse node, and _always_ running Goose
  on only that node, using `ON CLUSTER` for all DDL operations.

For the first case, run Goose separately for each node, using up-to limits to
ensure that all nodes complete a migration before any node begins the next
migration. Do not use `ON CLUSTER` for any DDL operations, otherwise the DDL
will replicate via ZooKeeper, but the Goose metadata table entry change will
_not_, so other nodes will not know they have applied the migration.

For the second case, the "DDL leader" node is the only node that tracks the
migration state. Other nodes are passive clients. All DDL must be run with `ON
CLUSTER` so that it is replicated via ZooKeeper across the whole ClickHouse
cluster. Consider setting
`distributed_ddl_output_mode=null_status_on_timeout_only_active` on the user
that Goose connects as so that lagging nodes do not return an error that will
cause Goose to re-queue the whole DDL. If the "DDL leader" node is lost and
another node must be promoted in its place, manual recovery steps would be
required to reconstruct the Goose migration state on the new DDL leader. Manual
reconstruction on fail-over can be worked around by manually creating the Goose
schema version table using the same steps as for the `Replicated` database
engine set out below.

In both cases, external co-ordination is required to prevent concurrent Goose
runs on the same database.

#### Clickhouse clustered with `Replicated` database engine

> [!WARNING] Manual creation of the goose db version table is required.

It is possible to run Goose against a ClickHouse cluster using the `Replicated`
databas engine _only_ by manually pre-creating the Goose db version table
_before_ running Goose for the first time.

If Goose is allowed to create `goose_db_version` with the `MergeTree` engine,
the table _definition_ will replicate automatically to all nodes, but table
_content_ will not be replicated when changed. So nodes will contain empty or
misleading `goose_db_version` tables, causing Goose to re-run past migrations
or worse.

Assuming the default table name `goose_db_version` and the `clickhouse` SQL
dialect, the table must be explicitly created by the administrator as:

```sql
CREATE TABLE IF NOT EXISTS goose_db_version (
    version_id Int64,
    is_applied UInt8,
    date Date default now(),
    tstamp DateTime default now()
)
ENGINE = ReplicatedMergeTree()
ORDER BY (version_id)
```

Note the `ENGINE = ReplicatedMergeTree()`.

Additionally, Goose should connect to ClickHouse only with a user that has the
settings `select_sequential_consistency=1`, `insert_quorum='auto'`, and
`insert_quorum_parallel=0` applied (directly or via a settings profile). It is
not sufficient to put this in migrations. This reduces - but does not
eliminate - the risk of races causing migrations to be re-run when sequential
Goose runs connect to different nodes before metadata changes have
replicated.

`select_sequential_consistency` only has an effect on `SELECT`s against data
written with `insert_quorum`; without also setting `insert_quorum='auto'` (and
`insert_quorum_parallel=0`, which sequential consistency requires) on the same
user, `select_sequential_consistency=1` does nothing for Goose's own
`INSERT`s. `insert_quorum='auto'` computes a majority quorum
(`number_of_replicas / 2 + 1`) automatically rather than requiring a hardcoded
replica count. E.g.:

```sql
CREATE USER goose
IDENTIFIED WITH sha256_password BY 'changeme'
SETTINGS
  select_sequential_consistency = 1,
  insert_quorum = 'auto',
  insert_quorum_parallel = 0;

GRANT ALL ON *.* TO goose WITH GRANT OPTION;

GRANT ACCESS MANAGEMENT ON *.* TO goose WITH GRANT OPTION;
```

Because these are set as a user/profile _default_ rather than built into
Goose's own queries, they also apply to any `INSERT` that a hand-written
migration runs directly - not just Goose's own metadata writes. A migration
that inserts a large volume of data, or inserts into a table under concurrent
application write load, should override this explicitly with `SETTINGS
insert_quorum=0` on its own statement; otherwise it inherits
`insert_quorum_timeout` (60s by default) as a new failure mode, on top of the
idempotency requirements described above. Overriding only works if the
settings profile leaves `insert_quorum` as a changeable default rather than
marking it `CONST`/`READONLY`.

Down-migrations use slower mutations. Goose's down-migration query sets
`mutations_sync = 2` directly on the `ALTER TABLE ... DELETE` statement it
issues, so the Goose process that issued it will not be told the delete
succeeded until the mutation has completed on all replicas. This only
narrows one failure mode: it stops *that* Goose invocation from declaring
success and moving on to another migration while the delete is still propagating. It does not stop a
*different* Goose invocation from seeing stale metadata while the mutation is
still in flight. It does not help if the issuing process crashes or its
connection drops before the mutation finishes confirming on all replicas,
as a retrying Goose instance may also see the stale metadata.
Goose then has no way to know whether the delete completed everywhere, same
as the general problem described in
[Requires idempotent DDL](#requires-idempotent-ddl) above. All replicas must
still be allowed time to finish applying a down-migration, and to be
confirmed as having done so, before another Goose invocation is started.

External co-ordination is still recommended to prevent concurrent Goose
executions or closely subsequent Goose executions against different nodes, as
Clickhouse lacks any cluster-wide mutual exclusion functionality.

## Appendix: `SYSTEM SYNC REPLICA` via a custom `Store`

`select_sequential_consistency=1` (above) is the generic fix for read-after-
write staleness: it needs no code, works for the `goose` CLI as-is, and
applies uniformly across deployment types.

[ClickHouse Cloud's own docs treat it as a fallback rather than a default](https://clickhouse.com/docs/cloud/reference/shared-merge-tree#consistency) but for Goose's case where a single read needs a Keeper round-trip
there's nothing much to choose from either way. However, they fail differently:
a read with `select_sequential_consistency=1` against a replica that hasn't
caught up fails immediately with `REPLICA_IS_NOT_IN_QUORUM`, expecting the
caller to retry against a different replica, while `SYSTEM SYNC REPLICA ...
LIGHTWEIGHT` blocks and waits for the current replica to catch up instead of
erroring. This blocking wait behaviour is preferable for a schema migration
tool.

An application that embeds Goose as a library (rather than the `goose` CLI) can
get this without modifying Goose itself, using the public
[`database.WithStore`](https://pkg.go.dev/github.com/pressly/goose/v3/database#WithStore)
option to wrap the built-in `clickhouse` store's read methods:

```go
type syncingStore struct {
    database.Store // embeds the built-in clickhouse Store; only reads are overridden
    tableName string
}

func (s *syncingStore) syncReplica(ctx context.Context, db database.DBTxConn) error {
    _, err := db.ExecContext(ctx, "SYSTEM SYNC REPLICA "+s.tableName+" LIGHTWEIGHT")
    return err
}

func (s *syncingStore) ListMigrations(ctx context.Context, db database.DBTxConn) ([]*database.ListMigrationsResult, error) {
    if err := s.syncReplica(ctx, db); err != nil {
        return nil, err
    }
    return s.Store.ListMigrations(ctx, db)
}

func (s *syncingStore) GetLatestVersion(ctx context.Context, db database.DBTxConn) (int64, error) {
    if err := s.syncReplica(ctx, db); err != nil {
        return -1, err
    }
    return s.Store.GetLatestVersion(ctx, db)
}

func (s *syncingStore) GetMigration(ctx context.Context, db database.DBTxConn, version int64) (*database.GetMigrationResult, error) {
    if err := s.syncReplica(ctx, db); err != nil {
        return nil, err
    }
    return s.Store.GetMigration(ctx, db, version)
}
```

This works because `Provider` pins a single `*sql.Conn` for the duration of
one `status`/`up`/`down` call and passes that same connection to every
`Store` method call in it - so the `SYSTEM SYNC REPLICA` statement and the
read that follows it land on the same node, whichever node that happens to
be. It requires the connecting user to have the `SYSTEM SYNC REPLICA`
privilege.

## Appendix: Relevant ClickHouse issues

Crash-safety, atomicity or idempotency of DDL:

* [ALTER ... UPDATE/DELETE reported as failed can already have committed and applied; retrying it double-applies the mutation (not idempotent under ambiguous Keeper outcome) #112133](https://github.com/ClickHouse/ClickHouse/issues/112133)
* [Acknowledged DROP TABLE / RENAME TABLE / EXCHANGE TABLES on an Atomic database can silently revert after power loss: the DDL commit record is never fsynced #111348](https://github.com/ClickHouse/ClickHouse/issues/111348)
* [DROP DATABASE is not crash-atomic: partial cascade leaves database with some tables removed #104665](https://github.com/ClickHouse/ClickHouse/issues/104665)
* [TRUNCATE TABLE leaves partial state after mid-rename crash on MergeTree #104624](https://github.com/ClickHouse/ClickHouse/issues/104624)
* [DROP PARTITION / DETACH PARTITION leaves partial state after mid-rename crash #104628](https://github.com/ClickHouse/ClickHouse/issues/104628)
* [REPLACE PARTITION FROM leaves partial state in destination after mid-rename crash #104660](https://github.com/ClickHouse/ClickHouse/issues/104660)
* [ReplicatedMergeTree replica becomes permanently readonly (Code 122) when power loss reverts an applied ALTER's local metadata commit; `force_restore_data` / `RESTART REPLICA` / `RESTORE REPLICA` all fail to recover #115549](https://github.com/ClickHouse/ClickHouse/issues/115549)
