# MySQL Go Migrations Example

This example demonstrates how to use Goose with MySQL for Go-based migrations.

## Prerequisites

- Go 1.21+
- MySQL/MariaDB server running
- A database created for testing

## Setup

### 1. Create a MySQL database

```sql
CREATE DATABASE goose_test CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'goose'@'%' IDENTIFIED BY 'password';
GRANT ALL PRIVILEGES ON goose_test.* TO 'goose'@'%';
FLUSH PRIVILEGES;
```

### 2. Build the custom binary

```bash
cd examples/mysql-go-migrations
go build -o goose-mysql *.go
```

This creates a `goose-mysql` binary with MySQL driver and your Go migrations compiled in.

## Usage

### Connection String

For MySQL, you **must** include these parameters:

- `parseTime=true` - Required for TIMESTAMP/DATETIME handling
- `multiStatements=true` - Required for multiple statements in a single migration

```bash
DBSTRING="user:password@tcp(localhost:3306)/dbname?parseTime=true&multiStatements=true"
```

### Run Migrations

```bash
# Check migration status
./goose-mysql mysql "$DBSTRING" status

# Apply all migrations
./goose-mysql mysql "$DBSTRING" up

# Check status again
./goose-mysql mysql "$DBSTRING" status

# Rollback one migration
./goose-mysql mysql "$DBSTRING" down

# Rollback all migrations
./goose-mysql mysql "$DBSTRING" down-to 0
```

### Expected Output

```bash
$ ./goose-mysql mysql "user:password@tcp(localhost:3306)/goose_test?parseTime=true&multiStatements=true" status
    Applied At                  Migration
    =======================================
    Pending                  -- 00001_create_users_table.sql
    Pending                  -- 00002_add_full_name.go
    Pending                  -- 00003_seed_users.go

$ ./goose-mysql mysql "user:password@tcp(localhost:3306)/goose_test?parseTime=true&multiStatements=true" up
    OK   00001_create_users_table.sql (2.45ms)
    OK   00002_add_full_name.go (1.12ms)
    OK   00003_seed_users.go (0.89ms)
    goose: no migrations to run. current version: 3

$ ./goose-mysql mysql "user:password@tcp(localhost:3306)/goose_test?parseTime=true&multiStatements=true" status
    Applied At                  Migration
    =======================================
    2024-01-15 10:30:45        -- 00001_create_users_table.sql
    2024-01-15 10:30:45        -- 00002_add_full_name.go
    2024-01-15 10:30:45        -- 00003_seed_users.go
```

## Key Points for MySQL

1. **parseTime=true** - Required for proper TIMESTAMP/DATETIME handling in Go
2. **multiStatements=true** - Required when you have multiple SQL statements separated by semicolons
3. **utf8mb4** - Use utf8mb4 character set for full Unicode support (including emojis)
4. **Go migrations** - Must be compiled into the custom binary (blank import pattern)

## Migration Files

| File | Type | Description |
|------|------|-------------|
| `00001_create_users_table.sql` | SQL | Creates the users table with MySQL-specific syntax |
| `00002_add_full_name.go` | Go | Adds a full_name column using ALTER TABLE |
| `00003_seed_users.go` | Go | Seeds initial users with MySQL's ON DUPLICATE KEY UPDATE |

## Best Practice: Separate Migrations Package

For production use, move migrations to a separate package:

```
myapp/
├── cmd/
│   └── goose-mysql/
│       └── main.go          # Import migrations package here
├── migrations/
│   ├── 00001_create_users_table.sql
│   ├── 00002_add_full_name.go
│   └── 00003_seed_users.go
└── go.mod
```

In `cmd/goose-mysql/main.go`:

```go
import (
    // Blank import to register migrations
    _ "github.com/myorg/myapp/migrations"
    "github.com/pressly/goose/v3"
    _ "github.com/go-sql-driver/mysql"
)
```

In migration files, change package from `main` to `migrations`:

```go
package migrations

import (
    "context"
    "database/sql"
    "github.com/pressly/goose/v3"
)

func init() {
    goose.AddMigrationContext(Up, Down)
}
```

## Troubleshooting

### "Go functions must be registered and built into a custom binary"

This error means you're trying to run Go migrations with the standard `goose` CLI instead of your custom binary. Make sure you:

1. Built the custom binary: `go build -o goose-mysql *.go`
2. Run the custom binary: `./goose-mysql mysql "..." up`

### "syntax error near unexpected token"

This usually means you're trying to run the binary as a shell script. Ensure you:
1. Built with `go build -o goose-mysql *.go`
2. Run with `./goose-mysql` (not `sh goose-mysql`)

### Connection refused / Access denied

Check:
- MySQL server is running
- User has correct privileges
- Host in connection string matches (localhost vs 127.0.0.1 vs %)
- Firewall allows connections on port 3306