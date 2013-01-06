# goose

goose is a database migration tool.

You can manage your database's evolution by creating incremental SQL or Go scripts.

# Install

    $ go get bitbucket.org/liamstask/goose

This will install the `goose` binary to your `$GOPATH/bin` directory.

# Usage

goose provides several commands to help manage your database schema.

## create

Create a new migration script.

    $ goose create AddSomeColumns
    $ goose: created db/migrations/20130106093224_AddSomeColumns.go

Edit the newly created script to define the behavior of your migration.

## up

Apply all available migrations.

    $ goose up
    $ goose: migrating db environment 'development', current version: 0, target: 3
    $ OK    001_basics.sql
    $ OK    002_next.sql
    $ OK    003_and_again.go

## down

Roll back a single migration from the current version.

    $ goose down
    $ goose: migrating db environment 'development', current version: 3, target: 2
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


`goose -h` provides more detailed info on each command.


# Migrations

goose supports migrations written in SQL or in Go.


## SQL Migrations

A sample SQL migration looks like:

    :::sql
    -- +goose Up
    CREATE TABLE post (
        id int NOT NULL,
        title text,
        body text,
        PRIMARY KEY(id)
    );

    -- +goose Down
    DROP TABLE post;

Notice the annotations in the comments. Any statements following `-- +goose Up` will be executed as part of a forward migration, and any statements following `-- +goose Down` will be executed as part of a rollback.


## Go Migrations

A sample Go migration looks like:

    :::go
    package migration_003

    import (
        "database/sql"
        "fmt"
    )

    func Up(txn *sql.Tx) {
        fmt.Println("Hello from migration_003 Up!")
    }

    func Down(txn *sql.Tx) {
        fmt.Println("Hello from migration_003 Down!")
    }

`Up()` will be executed as part of a forward migration, and `Down()` will be executed as part of a rollback.

A transaction is provided, rather than the DB instance directly, since goose also needs to record the schema version within the same transaction. Each migration should run as a single transaction to ensure DB integrity, so it's good practice anyway.


# Configuration

goose expects you to maintain a folder (typically called "db"), which contains the following:

* a dbconf.yml file that describes the database configurations you'd like to use
* a folder called "migrations" which contains .sql and/or .go scripts that implement your migrations

You may use the `-path` option to specify an alternate location for the folder containing your config and migrations.

A sample dbconf.yml looks like

    development:
        driver: postgres
        open: user=liam dbname=tester sslmode=disable

Here, `development` specifies the name of the environment, and the `driver` and `open` elements are passed directly to database/sql to access the specified database.

You may include as many environments as you like, and you can use the `-env` command line option to specify which one to use. goose defaults to using an environment called `development`.
