# Compiled Go migrations

## This example: Custom binary with compiled Go migrations

```bash
$ go build -o migrate main.go
```

```
$ ./migrate sqlite3 ./foo.db up

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
