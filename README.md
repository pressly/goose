# goose

goose is a database migration tool.

You can manage your database's evolution by creating incremental SQL or Go scripts.

# Install

    $ go get bitbucket.org/liamstask/goose

This will install the `goose` binary to your `$GOPATH/bin` directory.

# Usage

goose expects you to maintain a folder (typically called "db"), which contains the following:

* a dbconf.yml file that describes the database configurations you'd like to use
* a folder called "migrations" which contains .sql and/or .go scripts that implement your migrations

You may use the `--db` option to specify an alternate location for the folder containing your config and migrations.


# Migrations

goose supports migrations written in SQL or in Go.

## SQL Migrations

A sample SQL migration looks like:

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

## Database Configurations

A sample dbconf.yml looks like

	development:
    	driver: postgres
    	open: user=liam dbname=tester sslmode=disable

Here, `development` specifies the name of the configuration, and the `driver` and `open` elements are passed directly to database/sql to access the specified database.

You may include as many configurations as you like, and you can use the `--config` command line option to specify which one to use. goose defaults to using a configuration called `development`.
