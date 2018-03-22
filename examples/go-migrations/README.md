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

1. Move [main.go](main.go) into your `src/cmd/` directory

2. Adjust the imports to the paths for your project

3. Create `src/migrations/` directory with your migrations named `#######_migration_name.go` with the package declared as `migrations`.

4. Build the go package: `$ go build -o goose src/cmd/*.go`
