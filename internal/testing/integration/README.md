# Integration tests

This directory contains integration tests for the [pressly/goose/v3][goose_module] Go module. An
integration test is a test that runs against a real database (docker container) and exercises the
same driver used by the CLI.

## Why is this a separate module?

There are separate `go.mod` and `go.sum` files in this directory to allow for the use of different
dependencies. We leverage [multi-module workspaces](https://go.dev/doc/tutorial/workspaces) to glue
things together.

Namely, we want to avoid dependencies on docker, sql drivers, and other dependencies **that are**
not necessary for the core functionality of the goose library.

## Overview

There are separate migration files for each database that we support, see the [migrations
directory][migrations_dir]. Databases typically have different SQL syntax and features, so the
migration files are different.

A good set of migrations should be representative of the types of migrations users will write
typically write. This should include:

- Creating and dropping tables
- Adding and removing columns
- Creating and dropping indexes
- Inserting and deleting data
- Complex SQL statements that require special handling with `StatementBegin` and `StatementEnd`
  annotations
- Statements that must run outside a transaction, annotated with `-- +goose NO TRANSACTION`

There is a common test function that applies migrations up, down and then up again.

The gold standard is the PostgreSQL migration files. We try to make other migration files as close
to the PostgreSQL files as possible, but this is not always possible or desirable.

Lastly, some tests will assert for database state after migrations are applied.

To add a new `.sql` file, you can use the following command:

```
goose -s -dir testdata/migrations/<database_name> create <filename> sql
```

- Update the database name (e.g. `postgres`)
- Update the filename name (e.g. `b`) as needed

## Limitation

Note, the integration tests are not exhaustive.

They are meantto ensure that the goose library works with the various databases that we support and
the chosen drivers. We do not test every possible combination of operations, nor do we test every
possible edge case. We rely on the unit tests in the goose package to cover library-specific logic.

[goose_module]: https://pkg.go.dev/github.com/pressly/goose/v3
[migrations_dir]: ./testdata/migrations
