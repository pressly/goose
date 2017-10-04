# SQL + Go migrations

## This example: Best practice: Split migrations into a standalone package

```bash
$ go run main.go -dir ./db/migrations/ sqlite3 ./db/foo.db up
OK    00001_create_users_table.sql
OK    00002_rename_root.go
OK    00004_rename_admin.go
goose: no migrations to run. current version: 4
```
