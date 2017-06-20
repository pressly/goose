# SQL + Go migrations

## This example: Custom goose binary with built-in Go migrations

```bash
$ go build -o goose *.go
```

```
$ ./goose sqlite3 ./foo.db status
    Applied At                  Migration
    =======================================
    Pending                  -- 00001_create_users_table.sql
    Pending                  -- 00002_rename_root.go

$ ./goose sqlite3 ./foo.db up
OK    00001_create_users_table.sql
OK    00002_rename_root.go
goose: no migrations to run. current version: 2

$
    Applied At                  Migration
    =======================================
    Mon Jun 19 21:56:00 2017 -- 00001_create_users_table.sql
    Mon Jun 19 21:56:00 2017 -- 00002_rename_root.go
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
