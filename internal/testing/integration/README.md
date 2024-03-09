# Integration tests

This directory contains integration tests for the
[pressly/goose/v3](https://pkg.go.dev/github.com/pressly/goose/v3) Go module. There are separate
`go.mod` and `go.sum` files in this directory to allow for the use of different dependencies.

Namely, we want to avoid dependencies on docker, sql drivers, and other large dependencies that are
not necessary for the core functionality of the goose library.

The goal of the integration tests is to test the goose library and the various database drivers
against a real database. We use docker to start a database container and then run the tests against
that container.

There are separate migration files for each database that we support, see ./testdata/migrations
directory. Sometimes the migration files are the same, but sometimes they are different. This is
because different databases have different SQL syntax and features.

A good set of migrations should be representative of the types of migrations that users will write
in their own applications. This should include:

- Creating and dropping tables
- Adding and removing columns
- Creating and dropping indexes
- Inserting and deleting data
- Complex SQL statements that require special handling with `StatementBegin` and `StatementEnd`
  annotations

Each test should have an apply all up, rollback all down, and re-apply all up.

The gold standard is the PostgreSQL migration files. We try to make the other migration files as
close to the PostgreSQL files as possible.

To add a new sql file, you can use the following command:

- Update the database name (e.g. `postgres`)
- Update the migration name (e.g. `b`) as needed

```
goose -s -dir testdata/migrations/postgres create b sql
```

## Limitation

Note, the integration tests are not exhaustive.

They are meant to be a smoke test to ensure that the goose library works with the various databases
that we support. We do not test every possible combination of migration operations, nor do we test
every possible edge case. We rely on the unit tests in the `goose` package to cover those cases.
