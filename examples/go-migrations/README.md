# SQL + Go migrations

## This example: Custom goose binary with built-in Go migrations

```bash
$ go build -o goose-custom *.go
```

```bash
$ ./goose-custom sqlite3 ./foo.db status
    Applied At                  Migration
    =======================================
    Pending                  -- 00001_create_users_table.sql
    Pending                  -- 00002_rename_root.go
    Pending                  -- 00003_add_user_no_tx.go

$ ./goose-custom sqlite3 ./foo.db up
    OK   00001_create_users_table.sql (711.58µs)
    OK   00002_rename_root.go (302.08µs)
    OK   00003_add_user_no_tx.go (648.71µs)
    goose: no migrations to run. current version: 3

$ ./goose-custom sqlite3 ./foo.db status
    Applied At                  Migration
    =======================================
    00001_create_users_table.sql
    00002_rename_root.go
    00003_add_user_no_tx.go
```

## Best practice: Split migrations into a standalone package

1. Move [main.go](main.go) into your `cmd/` directory

2. Rename package name in all `*_.go` migration files from `main` to `migrations`.

3. Import this `migrations` package from your custom [cmd/main.go](main.go) file:

   ```go
   import (
       // Invoke init() functions within migrations pkg.
       _ "github.com/pressly/goose/example/migrations-go"
   )
   ```
