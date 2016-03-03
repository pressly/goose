# goose

Goose is a database migration tool. Manage your database's evolution by creating incremental SQL files or Go functions.

[![GoDoc Widget]][GoDoc] [![Travis Widget]][Travis]

This is a fork of https://bitbucket.org/liamstask/goose with following differences:
- No config files
- Meant to be imported by your application, so you can run complex Go migration functions with your own DB driver
- Standalone goose binary can only run SQL files -- we dropped building .go files in favor of the above

# Goals
- [x] Move lib/goose to top level directory
- [x] Remove all config files
- [x] Commands should be part of the API
- [x] Update & finish README
- [ ] Registry for Go migration functions

# Install

    $ go get -u github.com/pressly/cmd/goose

This will install the `goose` binary to your `$GOPATH/bin` directory.

*Note: A standalone goose binary can only run pure SQL migrations. To run complex Go migrations, you have to import this pkg and register functions.*

# Usage

`goose [OPTIONS] DRIVER DBSTRING COMMAND`

Examples:

    $ goose postgres "user=postgres dbname=postgres sslmode=disable" up
    $ goose mysql "user:password@/dbname" down
    $ goose sqlite3 ./foo.db status

## create

Create a new Go migration.

    $ goose create AddSomeColumns
    $ goose: created db/migrations/20130106093224_AddSomeColumns.go

Edit the newly created script to define the behavior of your migration.

You can also create an SQL migration:

    $ goose create AddSomeColumns sql
    $ goose: created db/migrations/20130106093224_AddSomeColumns.sql

## up

Apply all available migrations.

    $ goose up
    $ goose: migrating db environment 'development', current version: 0, target: 3
    $ OK    001_basics.sql
    $ OK    002_next.sql
    $ OK    003_and_again.go

### option: pgschema

Use the `pgschema` flag with the `up` command specify a postgres schema.

    $ goose -pgschema=my_schema_name up
    $ goose: migrating db environment 'development', current version: 0, target: 3
    $ OK    001_basics.sql
    $ OK    002_next.sql
    $ OK    003_and_again.go

## down

Roll back a single migration from the current version.

    $ goose down
    $ goose: migrating db environment 'development', current version: 3, target: 2
    $ OK    003_and_again.go

## redo

Roll back the most recently applied migration, then run it again.

    $ goose redo
    $ goose: migrating db environment 'development', current version: 3, target: 2
    $ OK    003_and_again.go
    $ goose: migrating db environment 'development', current version: 2, target: 3
    $ OK    003_and_again.go

## status

Print the status of all migrations:

    $ goose status
    $ goose: status for environment 'development'
    $   Applied At                  Migration
    $   =======================================
    $   Sun Jan  6 11:25:03 2013 -- 001_basics.sql
    $   Sun Jan  6 11:25:03 2013 -- 002_next.sql
    $   Pending                  -- 003_and_again.go

## dbversion

Print the current version of the database:

    $ goose dbversion
    $ goose: dbversion 002


`goose -h` provides more detailed info on each command.


# Migrations

goose supports migrations written in SQL or in Go - see the `goose create` command above for details on how to generate them.

## SQL Migrations

A sample SQL migration looks like:

```sql
-- +goose Up
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE post;
```

Notice the annotations in the comments. Any statements following `-- +goose Up` will be executed as part of a forward migration, and any statements following `-- +goose Down` will be executed as part of a rollback.

By default, SQL statements are delimited by semicolons - in fact, query statements must end with a semicolon to be properly recognized by goose.

More complex statements (PL/pgSQL) that have semicolons within them must be annotated with `-- +goose StatementBegin` and `-- +goose StatementEnd` to be properly recognized. For example:

```sql
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
BEGIN
  FOR create_query IN SELECT
      'CREATE TABLE IF NOT EXISTS histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( CHECK( created_at >= timestamp '''
      || TO_CHAR( d, 'YYYY-MM-DD 00:00:00' )
      || ''' AND created_at < timestamp '''
      || TO_CHAR( d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00' )
      || ''' ) ) inherits ( histories );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
  END LOOP;  -- LOOP END
END;         -- FUNCTION END
$$
language plpgsql;
-- +goose StatementEnd
```

## Go Migrations

A sample Go migration looks like:

```go
package main

import (
    "database/sql"
    "fmt"
)

func Up_20130106222315(txn *sql.Tx) {
    fmt.Println("Hello from migration 20130106222315 Up!")
}

func Down_20130106222315(txn *sql.Tx) {
    fmt.Println("Hello from migration 20130106222315 Down!")
}
```

`Up_20130106222315()` will be executed as part of a forward migration, and `Down_20130106222315()` will be executed as part of a rollback.

The numeric portion of the function name (`20130106222315`) must be the leading portion of migration's filename, such as `20130106222315_descriptive_name.go`. `goose create` does this by default.

A transaction is provided, rather than the DB instance directly, since goose also needs to record the schema version within the same transaction. Each migration should run as a single transaction to ensure DB integrity, so it's good practice anyway.

## License

Licensed under [MIT License](./LICENSE)

[GoDoc]: https://godoc.org/github.com/pressly/goose
[GoDoc Widget]: https://godoc.org/github.com/pressly/goose?status.svg
[Travis]: https://travis-ci.org/pressly/goose
[Travis Widget]: https://travis-ci.org/pressly/goose.svg?branch=master
