# SQL + Go migrations

## This example: Best practice: Split migrations into a standalone package

```bash
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db up-missing
OK    00001_create_users_table.sql
OK    00002_rename_root.go
OK    00004_rename_admin.go
```
Remove "_" at _00003 and _00005 migrations, migrate again:
```bash
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db up-missing
OK    00003_rename_admin.go
OK    00005_rename_admin.go
```
