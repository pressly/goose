# SQL + Go migrations

## Example custom goose binary with built-in Go migrations

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

2. Rename migration functions to `migrations` pkg

3. Import `migrations` package from [cmd/main.go](main.go)

    ```go
    import (
        _ "github.com/pressly/goose/example/migrations-go"
    )
    ```

    This will cause all `init()` functions to be called within `migrations` pkg, thus registering the migration functions properly.
