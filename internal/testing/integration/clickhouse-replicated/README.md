# clickhouse-replicated cluster configs

XML fragments describing a two-node ClickHouse cluster (`ch1`, `ch2`) with an
embedded ClickHouse Keeper on `ch1`. Used to exercise the goose
`clickhouse-replicated` dialect against a real replicated cluster.

- `ch1.xml` / `ch2.xml` — cluster / `remote_servers` declaration for each node.
- `macros-ch1.xml` / `macros-ch2.xml` — per-node `{shard}` / `{replica}` macros.
- `keeper.xml` — embedded Keeper server config (used by `ch1`).
- `keeper-client.xml` — Keeper client config (used by `ch2`).
- `users.xml` — permissive `default` user for the test fixture.

These files are bind-mounted read-only into the containers launched by
`internal/testing/testdb.NewClickHouseReplicated`, which uses
[`ory/dockertest`] to bring the cluster up on a private user network for the
duration of the `TestClickhouseReplicated` integration test. No `docker
compose` is required.

[`ory/dockertest`]: https://github.com/ory/dockertest
