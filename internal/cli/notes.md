```

GOOSE_DB=
GOOSE_DBSTRING=
GOOSE_MIGRATION_DIR=

EXAMPLES

goose --db="" --dbstring="" -dir=="" up

Or set environment variables:

GOOSE_DB=postgres
GOOSE_DBSTRING="postgres://dbuser:password1@localhost/testdb?sslmode=disable"
GOOSE_DIR=./data/schema/migrations

goose up

USAGE
  goose [root flags] <command> <subcommand> [flags]

ROOT FLAGS
  --help      Show help for command
  --version   Show goose CLI version
  --db        The database dialect, see SUPPORTED DATABASES (default: postgres)
  --dbstring  The database connection string
  --dir       The directory containing migration files (default: ./migrations)

SUPPORTED DATABASES
    postgres
    mysql
    sqlite3
    mssql
    redshift
    tidb
    clickhouse
    vertica

CORE COMMANDS
    up          Migrate the database to most recent version
    up-by-one   Migrate exactly one migration up
    up-to       Migrate up to, and including, the specified version
    down        Migrate down the most recent version
    down-to     Migrtae down to, but not including, the specified version
    redo        Re-run the most recently run migration
    reset       Rollback all migrations
    status      List the migrations and their status
    version     Print the current version of the database

ADDITIONAL COMMANDS
    create
    fix
    validate
    init


CORE COMMANDS
  auth:        Authenticate gh and git with GitHub
  browse:      Open the repository in the browser
  codespace:   Connect to and manage codespaces
  gist:        Manage gists
  issue:       Manage issues
  pr:          Manage pull requests
  release:     Manage releases
  repo:        Manage repositories
```
