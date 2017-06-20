# SQL migrations only

See [this example](../go-migrations) for Go migrations.

```bash
$ go get -u github.com/pressly/goose/cmd/goose
```

```bash
$ goose sqlite3 ./foo.db status
    Applied At                  Migration
    =======================================
    Pending                  -- 00001_create_users_table.sql
    Pending                  -- 00002_rename_root.sql

$ goose sqlite3 ./foo.db up
OK    00001_create_users_table.sql
OK    00002_rename_root.sql
goose: no migrations to run. current version: 2

$ goose sqlite3 ./foo.db status
    Applied At                  Migration
    =======================================
    Mon Jun 19 21:56:00 2017 -- 00001_create_users_table.sql
    Mon Jun 19 21:56:00 2017 -- 00002_rename_root.sql
```
