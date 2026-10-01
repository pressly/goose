# Contributing

Thanks for contributing to goose.

## Development

```bash
go test ./...
```

Integration tests that need Docker live under `internal/testing/integration`. See that package's README for how to run them.

## Adding a new SQL dialect

Most of the time you do **not** need a first-class dialect in this repository. goose already supports [custom stores](https://pressly.github.io/goose/documentation/custom-store/) and a custom [`database/dialect.Querier`](./database/dialect/querier.go) via [`database.NewStoreFromQuerier`](./database/dialects.go). Prefer that path when the dialect is niche, experimental, or only needed by your application.

If a dialect should ship in goose itself (CLI driver + library constant + tests), use the checklist below. Recent examples: Turso ([#658](https://github.com/pressly/goose/pull/658)) and YDB ([#592](https://github.com/pressly/goose/pull/592)).

### Checklist

1. **Querier**
   - Add `New<Name>()` in [`internal/dialects/`](./internal/dialects/) implementing [`dialect.Querier`](./database/dialect/querier.go) (and optionally [`QuerierExtender`](./database/dialect/querier_extended.go) for `TableExists`).
   - Reuse an existing dialect via embedding when SQL is compatible (see [`turso.go`](./internal/dialects/turso.go) embedding `sqlite3`).

2. **Library registration**
   - Add a `Dialect` constant in [`database/dialects.go`](./database/dialects.go) and map it in `NewStore`.
   - Mirror the constant / `SetDialect` string aliases in [`dialect.go`](./dialect.go) and any switch in [`db.go`](./db.go) / [`internal/legacystore`](./internal/legacystore) that still selects queriers by dialect.
   - Export the alias from the root `goose` package if other dialects do.

3. **CLI driver (optional but usual for official dialects)**
   - Add `cmd/goose/driver_<name>.go` with a `//go:build !no_<tag>` blank import of the database driver (see [`driver_turso.go`](./cmd/goose/driver_turso.go)).
   - Document the driver name and example DSN in the CLI help text in [`cmd/goose/main.go`](./cmd/goose/main.go).
   - Add the driver dependency to `go.mod` when required.
   - Mention the `no_<tag>` build tag next to the others in the README install section if applicable.

4. **Tests**
   - Unit-level store behavior is covered by [`database/store_test.go`](./database/store_test.go) patterns.
   - Prefer an integration test under `internal/testing/integration` with a helper in `internal/testing/testdb` when a container image exists.
   - Add dialect-specific migration fixtures under `testdata/migrations/<dialect>` when SQL differs.

5. **Docs**
   - List the dialect in the README driver/usage sections.
   - Note the change in `CHANGELOG.md`.

Open a PR against `main` with the checklist items that apply. Keep the first commit focused on the querier + registration; CLI/driver and docs can be separate commits when that keeps review easier.
