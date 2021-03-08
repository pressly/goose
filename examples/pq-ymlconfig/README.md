# PostgresSQL + YML config migration

## This example: Custom goose binary for PostgreSQL migrations with YML config.

```bash
$ go build -o goose *.go
```

```bash
$ ./goose -h
```

## Best practice: Split migrations into a standalone package

1. Move [main.go](main.go) into your `cmd/` directory
