# goose

Goose is a database migration tool. Manage your database schema by creating incremental SQL changes or Go functions.

[![Build Status](https://travis-ci.org/webconnex/goose.svg?branch=master)](https://travis-ci.org/webconnex/goose)

# Install

    $ go get -u github.com/webconnex/goose/cmd/goose

This will install the `goose` binary to your `$GOPATH/bin` directory.

# Usage

```
Usage: goose [OPTIONS] DRIVER DBSTRING COMMAND

Drivers:
    postgres
    mysql
    sqlite3
    redshift

Commands:
    up                   Migrate the DB to the most recent version available
    up-to VERSION        Migrate the DB to a specific VERSION
    apply VERSION        Applies a single VERSION to the DB
    revert VERSION       Reverts a single VERSION to the DB
    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION
    redo                 Re-run the latest migration
    status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with next version

Options:
    -dir string
        directory with migration files (default "./migrations/")

Examples:
    goose sqlite3 ./foo.db status
    goose sqlite3 ./foo.db create init sql
    goose sqlite3 ./foo.db create add_some_column sql
    goose sqlite3 ./foo.db create fetch_user_data go
    goose sqlite3 ./foo.db up

    goose mysql "user:password@tcp(<path to mysql>localhost:3306)/webconnex?parseTime=true" status
    goose postgres "user=postgres dbname=postgres sslmode=disable" status
    goose redshift "postgres://user:password@qwerty.us-east-1.redshift.amazonaws.com:5439/db" status
```
## create

Create a new SQL migration.

    $ goose create add_some_column sql
    $ Created new file: 00001_add_some_column.sql

Edit the newly created file to define the behavior of your migration.

You can also create a Go migration, if you then invoke it with [your own goose binary](#go-migrations):

    $ goose create fetch_user_data go
    $ Created new file: 00002_fetch_user_data.go

## up

Apply all available migrations.

    $ goose up
    $ goose: migrating db environment 'development', current version: 0, target: 3
    $ OK    001_basics.sql
    $ OK    002_next.sql
    $ OK    003_and_again.go
    $ OK    005_and_again_and_again.go

## up-to

Migrate up to a specific version.

    $ goose up-to 20170506082420
    $ OK    20170506082420_create_table.sql

## apply

Applies a single VERSION to the DB.

    $ goose apply 20170506082420
    $ OK    20170506082420_create_table.sql

## down

Roll back a single migration from the current version.

    $ goose down
    $ goose: migrating db environment 'development', current version: 3, target: 2
    $ OK    003_and_again.go

## down-to

Roll back migrations to a specific version.

    $ goose down-to 20170506082527
    $ OK    20170506082527_alter_column.sql

## redo

Roll back the most recently applied migration, then run it again.

    $ goose redo
    $ OK    003_and_again.sql
    $ OK    003_and_again.sql

## status

Print the status of all migrations:

    $ goose status
    $ goose: status for environment 'development'
    $   Applied At                  Migration
    $   =======================================
    $   Sun Jan  6 11:25:03 2013 -- 001_basics.sql
    $   Sun Jan  6 11:25:03 2013 -- 002_next.sql
    $   Pending                  -- 003_and_again.go
    $   Sun Jan  7 11:25:03 2013 -- 004_next.sql

Note: for MySQL [parseTime flag](https://github.com/go-sql-driver/mysql#parsetime) must be enabled.

## version

Print the current version of the database:

    $ goose version
    $ goose: version 002

# Migrations

goose supports migrations written in SQL or in Go.

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

By default, all migrations are run within a transaction. Some statements like `CREATE DATABASE`, however, cannot be run within a transaction. You may optionally add `-- +goose NO TRANSACTION` to the top of your migration
file in order to skip transactions within that specific migration file. Both Up and Down migrations within this file will be run without transactions.

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

1. Create your own goose binary, see [example](./examples/go-migrations)
2. Import `github.com/webconnex/goose`
3. Register your migration functions
4. Run goose command, ie. `goose.Up(db *sql.DB, dir string)`

A [sample Go migration 00002_users_add_email.go file](./example/migrations-go/00002_rename_root.go) looks like:

```go
package migrations

import (
	"database/sql"

	"github.com/webconnex/goose"
)

func init() {
	goose.AddMigration(Up, Down)
}

func Up(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='admin' WHERE username='root';")
	if err != nil {
		return err
	}
	return nil
}

func Down(tx *sql.Tx) error {
	_, err := tx.Exec("UPDATE users SET username='root' WHERE username='admin';")
	if err != nil {
		return err
	}
	return nil
}
```

## License

Licensed under [MIT License](./LICENSE)
