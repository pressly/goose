# goose

Gander is a database migration tool. Manage your database schema by creating incremental SQL changes or Go functions.

This is a new fort with work still to be done

### Goals of this fork

`github.com/geniusmonkey/gander` is a fork of `github.com/pressly/goose` which is a fork of `bitbucket.org/liamstask/goose` with the following changes:
- TOML config file support multipule environments
- [Default gander binary](./cmd/gander/main.go) can migrate SQL files only
- Baseline migrations for existing databases
- Migrate CLI to use cobra style commands
- Update exposed API in package `github.com/geniusmonkey/gander` to do more with less
    - Eliminate the need to set a dialect prior to calling the api
    - Introduce adding new drivers/dialects with without updating core library
    - Seperate the CLI specific logic from core API
- Imporove upon versioning
    - Maintaine current timestamped based default versions
    - Add more statagies for versioning via the CLI
- Imporove logging
    - Allow verbose logging of sql statements
    - Better status updates files are applied


# Install

    $ go get -u github.com/geniusmonkey/gander/cmd/gander

This will install the `gander` binary to your `$GOPATH/bin` directory.


# Usage

```
CLI for running SQL migrations

Usage:
  gander [command]

Available Commands:
  baseline    Baseline an existing db to a specific VERSION
  create      Creates new migration file with the current timestamp
  down        Roll back the version by 1
  help        Help about any command
  redo        Re-run the latest migration
  status      Dump the migration status for the current DB
  up          Migrate the DB to the most recent version available
  version     Print the current version of the database

Flags:
  -c, --config string   config file location (default dbconf.toml)
      --dir string      directory containing the migration files (default "./migrations")
      --driver string   name of the database driver
      --dsn string      dataSourceName to connect to the server
  -e, --env string      name of the environment to use (default "development")
  -h, --help            help for gander

Use "gander [command] --help" for more information about a command.
```
## create

Create a new SQL migration.

    $ gander create add_some_column 
    $ Created new file: 20170506082420_add_some_column.sql

Edit the newly created file to define the behavior of your migration.

You can also create a Go migration, if you then invoke it with [your own gander binary](#go-migrations):

    $ gander create fetch_user_data --type=go
    $ Created new file: 20170506082421_fetch_user_data.go

## up

Apply all available migrations.

    $ gander up
    $ gander: migrating db environment 'development', current version: 0, target: 3
    $ OK    001_basics.sql
    $ OK    002_next.sql
    $ OK    003_and_again.go

Migrate up to a specific version.

    $ gander up --to=20170506082420
    $ OK    20170506082420_create_table.sql

## down

Roll back a single migration from the current version.

    $ gander down
    $ gander: migrating db environment 'development', current version: 3, target: 2
    $ OK    003_and_again.go

Roll back migrations to a specific version.

    $ gander down --to=20170506082527
    $ OK    20170506082527_alter_column.sql

## redo

Roll back the most recently applied migration, then run it again.

    $ gander redo
    $ gander: migrating db environment 'development', current version: 3, target: 2
    $ OK    003_and_again.go
    $ gander: migrating db environment 'development', current version: 2, target: 3
    $ OK    003_and_again.go

## status

Print the status of all migrations:

    $ gander status
    $ gander: status for environment 'development'
    $   Applied At                  Migration
    $   =======================================
    $   Sun Jan  6 11:25:03 2013 -- 001_basics.sql
    $   Sun Jan  6 11:25:03 2013 -- 002_next.sql
    $   Pending                  -- 003_and_again.go

Note: for MySQL [parseTime flag](https://github.com/go-sql-driver/mysql#parsetime) must be enabled.

## version

Print the current version of the database:

    $ gander version
    $ gander: version 002

## baseline

Mark some migrations as applied for migrating to gander on existing databases

    $ gander baseline 003
    $ OK    001_basics.sql
    $ OK    002_next.sql
    $ OK    003_and_again.go

# Config File

By default the CLI will look for a `dbconf.toml` relative to the current directory. By default if no `env` flag is passed it will use `development` as the default environment. You can use the `config` flag to specify the location of the config file. `migrationsDir` is relative to the location of the `dbconf.toml` file.

    [env.development]
    dsn = "user=root password=supersupersecret host=127.0.0.1 port=9001 dbname=dev-monkey"
    migrationsDir = "migrations"
    driver = "redshift"

    [env.production]
    dsn = "user=root password=supersupersecret host=127.0.0.1 port=9001 dbname=prod-monkey"
    migrationsDir = "migrations"
    driver = "redshift"

# Migrations

gander supports migrations written in SQL or in Go.

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

1. Create your own gander binary, see [example](./examples/go-migrations)
2. Import `github.com/geniusmonkey/gander`
3. Register your migration functions
4. Run gander command, ie. `Up(db *sql.DB, dir string)`

A [sample Go migration 00002_users_add_email.go file](./example/migrations-go/00002_rename_root.go) looks like:

```go
package migrations

import (
	"database/sql"

	"github.com/geniusmonkey/gander"
)

func init() {
	AddMigration(Up, Down)
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

# Hybrid Versioning
Please, read the [versioning problem](https://github.com/pressly/goose/issues/63#issuecomment-428681694) first.

We strongly recommend adopting a hybrid versioning approach, using both timestamps and sequential numbers. Migrations created during the development process are timestamped and sequential versions are ran on production. We believe this method will prevent the problem of conflicting versions when writing software in a team environment.

To help you adopt this approach, `create` will use the current timestamp as the migration version. When you're ready to deploy your migrations in a production environment, we also provide a helpful `fix` command to convert your migrations into sequential order, while preserving the timestamp ordering. We recommend running `fix` in the CI pipeline, and only when the migrations are ready for production.

## License

Licensed under [MIT License](./LICENSE)

[GoDoc]: https://godoc.org/github.com/pressly/goose
[GoDoc Widget]: https://godoc.org/github.com/pressly/goose?status.svg
[Travis]: https://travis-ci.org/pressly/goose
[Travis Widget]: https://travis-ci.org/pressly/goose.svg?branch=master
