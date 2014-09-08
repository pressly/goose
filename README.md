# goose

goose is a database migration tool.

You can manage your database's evolution by creating incremental SQL or Go scripts.

[![Build Status](https://drone.io/bitbucket.org/liamstask/goose/status.png)](https://drone.io/bitbucket.org/liamstask/goose/latest)

# Install

    $ go get bitbucket.org/liamstask/goose/cmd/goose

This will install the `goose` binary to your `$GOPATH/bin` directory.

You can also build goose into your own applications by importing `bitbucket.org/liamstask/goose/lib/goose`. Documentation is available at [godoc.org](http://godoc.org/bitbucket.org/liamstask/goose/lib/goose).

NOTE: the API is still new, and may undergo some changes.

# Usage

goose provides several commands to help manage your database schema.

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


# Configuration

goose expects you to maintain a folder (typically called "db"), which contains the following:

* a `dbconf.yml` file that describes the database configurations you'd like to use
* a folder called "migrations" which contains `.sql` and/or `.go` scripts that implement your migrations

You may use the `-path` option to specify an alternate location for the folder containing your config and migrations.

A sample `dbconf.yml` looks like

```yml
development:
    driver: postgres
    open: user=liam dbname=tester sslmode=disable
```

Here, `development` specifies the name of the environment, and the `driver` and `open` elements are passed directly to database/sql to access the specified database.

You may include as many environments as you like, and you can use the `-env` command line option to specify which one to use. goose defaults to using an environment called `development`.

goose will expand environment variables in the `open` element. For an example, see the Heroku section below.

## Other Drivers
goose knows about some common SQL drivers, but it can still be used to run Go-based migrations with any driver supported by `database/sql`. An import path and known dialect are required.

Currently, available dialects are: "postgres", "mysql", or "sqlite3"

To run Go-based migrations with another driver, specify its import path and dialect, as shown below.

```yml
customdriver:
    driver: custom
    open: custom open string
    import: github.com/custom/driver
    dialect: mysql
```

NOTE: Because migrations written in SQL are executed directly by the goose binary, only drivers compiled into goose may be used for these migrations.

## Using goose with Heroku

These instructions assume that you're using [Keith Rarick's Heroku Go buildpack](https://github.com/kr/heroku-buildpack-go). First, add a file to your project called (e.g.) `install_goose.go` to trigger building of the goose executable during deployment, with these contents:

```go
// use build constraints to work around http://code.google.com/p/go/issues/detail?id=4210
// +build heroku

// note: need at least one blank line after build constraint
package main

import _ "bitbucket.org/liamstask/goose/cmd/goose"
```

[Set up your Heroku database(s) as usual.](https://devcenter.heroku.com/articles/heroku-postgresql)

Then make use of environment variable expansion in your `dbconf.yml`:

```yml
production:
    driver: postgres
    open: $DATABASE_URL
```

To run goose in production, use `heroku run`:

    heroku run goose -env production up

# Contributors

Thank you!

* Josh Bleecher Snyder (josharian)
* Abigail Walthall (ghthor)
* Daniel Heath (danielrheath)
* Chris Baynes (chris_baynes)
* Michael Gerow (gerow)
* Vytautas Šaltenis (rtfb)
* James Cooper (coopernurse)
* Gyepi Sam (gyepisam)
* Matt Sherman (clipperhouse)
* runner_mei
* John Luebs (jkl1337)
* Luke Hutton (lukehutton)
* Kevin Gorjan (kevingorjan)
* Brendan Fosberry (Fozz)
* Nate Guerin (gusennan)
