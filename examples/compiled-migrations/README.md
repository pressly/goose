# Compiled Go migrations

## This example: Custom binary with compiled Go migrations


You can migrate all of your generate `*.go` migrations by calling `goose.Registered().Up(...)`. This function performs all of the Go migrations that were registered with `goose.AddMigration()`, which is called in the `init` block of generate Go migration files.

### Example 
```bash
$ go build -o migrate main.go
```

```
$ ./migrate sqlite3 ./foo.db up
OK    00002_rename_root.go
goose: no migrations to run. current version: 1
```
