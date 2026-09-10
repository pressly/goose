# clickhouse-replicated integration compose

Two-node ClickHouse cluster (`ch1`, `ch2`) with an embedded ClickHouse Keeper
on `ch1`. Used to exercise the goose `clickhouse-replicated` dialect against
a real replicated cluster. **Not wired into CI.**

## Bring it up

```
docker compose up -d
docker compose ps
```

- `ch1`: TCP `9000`, HTTP `8123`
- `ch2`: TCP `9001`, HTTP `8124`

Cluster name: `goose_cluster`. Macros `{shard}` and `{replica}` are configured
on each server so `ReplicatedMergeTree(...)` can rely on the default replica
path / name (no need to pass explicit ZK path / replica args).

## Run migrations

```
GOOSE_CLICKHOUSE_CLUSTER=goose_cluster \
  goose -dir ./your-migrations \
        clickhouse-replicated \
        "clickhouse://default@localhost:9000/default" up
```

Verify the same data is visible on both replicas:

```
docker exec goose-ch1 clickhouse-client --query \
  "SELECT * FROM goose_db_version ORDER BY version_id"
docker exec goose-ch2 clickhouse-client --query \
  "SELECT * FROM goose_db_version ORDER BY version_id"
```

## Tear down

```
docker compose down -v
```
