# SQL + Go migrations

## This example: Best practice: Split migrations into a standalone package

Also take a look at "missing" migrations handling.

```bash
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db status
    Applied At                  Migration
    =======================================
    Pending                  -- 00001_create_users_table.sql
    Pending                  -- 00002_rename_root.go
    Pending                  -- 00004_rename_admin.go
$ go run main.go -dir ./db/migrations -missing-only sqlite3 ./db/foo.db status
    Missing migrations
    ===========
    00001_create_users_table.sql
    00002_rename_root.go
    00004_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db up
OK    00001_create_users_table.sql
OK    00002_rename_root.go
OK    00004_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db status
    Applied At                  Migration
    =======================================
    Fri Oct  6 13:03:09 2017 -- 00001_create_users_table.sql
    Fri Oct  6 13:03:09 2017 -- 00002_rename_root.go
    Fri Oct  6 13:03:09 2017 -- 00004_rename_admin.go
$ go run main.go -dir ./db/migrations -missing-only sqlite3 ./db/foo.db status
goose: no missing migrations

```
Get "missing" migrations: remove "_" at _00003 and _00005 migrations, then do:
```bash
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db status
    Applied At                  Migration
    =======================================
    Fri Oct  6 13:03:09 2017 -- 00001_create_users_table.sql
    Fri Oct  6 13:03:09 2017 -- 00002_rename_root.go
    Pending                  -- 00003_rename_admin.go
    Fri Oct  6 13:03:09 2017 -- 00004_rename_admin.go
    Pending                  -- 00005_rename_admin.go
$ go run main.go -dir ./db/migrations -missing-only sqlite3 ./db/foo.db status
    Missing migrations
    ===========
    00003_rename_admin.go
    00005_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db up-by-one
OK    00003_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db up-by-one
OK    00005_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db status
    Applied At                  Migration
    =======================================
    Fri Oct  6 13:03:09 2017 -- 00001_create_users_table.sql
    Fri Oct  6 13:03:09 2017 -- 00002_rename_root.go
    Fri Oct  6 13:20:52 2017 -- 00003_rename_admin.go
    Fri Oct  6 13:03:09 2017 -- 00004_rename_admin.go
    Fri Oct  6 13:20:56 2017 -- 00005_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db down
OK    00005_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db down
OK    00003_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db down
OK    00004_rename_admin.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db down
OK    00002_rename_root.go
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db down
OK    00001_create_users_table.sql
$ go run main.go -dir ./db/migrations sqlite3 ./db/foo.db down
2017/10/06 16:21:27 goose run: no migration 0
exit status 1
```
