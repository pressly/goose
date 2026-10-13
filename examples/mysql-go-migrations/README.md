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

For MySQL, the following DSN parameters are recommended:

| Parameter | Required | Description |
|-----------|----------|-------------|
| `parseTime=true` | **Yes** | Required for proper `TIMESTAMP`/`DATETIME` handling in Go. Without this, time values are returned as `[]byte` instead of `time.Time`. |
| `multiStatements=true` | **Yes** | Required when SQL migrations contain multiple statements separated by semicolons. Also allows Goose to execute batch statements. |
| `charset=utf8mb4` | Recommended | Ensures full Unicode support (including emojis). |
| `collation=utf8mb4_unicode_ci` | Recommended | Sets default collation for new tables. |

Example DSN using the test database created above:

```bash
DBSTRING="goose:password@tcp(localhost:3306)/goose_test?parseTime=true&multiStatements=true&charset=utf8mb4&collation=utf8mb4_unicode_ci"
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
$ ./goose-mysql mysql "$DBSTRING" status
    Applied At                  Migration
    =======================================
    Pending                  -- 00001_create_users_table.sql
    Pending                  -- 00002_add_full_name.go
    Pending                  -- 00003_seed_users.go

$ ./goose-mysql mysql "$DBSTRING" up
    OK   00001_create_users_table.sql (2.45ms)
    OK   00002_add_full_name.go (1.12ms)
    OK   00003_seed_users.go (0.89ms)
    goose: no migrations to run. current version: 3

$ ./goose-mysql mysql "$DBSTRING" status
    Applied At                  Migration
    =======================================
    2024-01-15 10:30:45        -- 00001_create_users_table.sql
    2024-01-15 10:30:45        -- 00002_add_full_name.go
    2024-01-15 10:30:45        -- 00003_seed_users.go
```

## Key Points for MySQL

1. **parseTime=true** (Required) - Enables proper `TIMESTAMP`/`DATETIME` handling. Without this, Go's `database/sql` returns time values as `[]byte` instead of `time.Time`.

2. **multiStatements=true** (Required) - Allows multiple SQL statements in a single query. Required for:
   - SQL migrations with multiple statements separated by semicolons
   - Goose's internal batch operations

3. **utf8mb4** (Recommended) - Use `utf8mb4` character set for full Unicode support (including emojis). Set via DSN or table definition.

4. **DDL and Transactions** - MySQL DDL statements (`CREATE TABLE`, `ALTER TABLE`, `DROP TABLE`, etc.) cause **implicit commits** and cannot be rolled back inside a transaction. Use:
   - `-- +goose NO TRANSACTION` annotation for SQL migrations
   - `goose.AddMigrationNoTxContext` for Go migrations (receives `*sql.DB` instead of `*sql.Tx`)

5. **Go migrations** - Must be compiled into the custom binary (blank import pattern).

## Migration Files

| File | Type | Transaction | Description |
|------|------|-------------|-------------|
| `00001_create_users_table.sql` | SQL | No (`NO TRANSACTION`) | Creates the users table with MySQL-specific syntax |
| `00002_add_full_name.go` | Go | No (`AddMigrationNoTxContext`) | Adds a full_name column using ALTER TABLE (DDL) |
| `00003_seed_users.go` | Go | Yes (`AddMigrationContext`) | Seeds initial users with INSERT/DELETE (DML) |

Note: Migration 00003 uses `AddMigrationContext` (with transaction) because `INSERT` and `DELETE` are DML operations that support rollback. Migrations 00001 and 00002 use no-transaction mode because they contain DDL.

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
    // Use AddMigrationNoTxContext for DDL (CREATE/ALTER/DROP TABLE)
    goose.AddMigrationNoTxContext(Up, Down)
    
    // Use AddMigrationContext for DML (INSERT/UPDATE/DELETE)
    // goose.AddMigrationContext(Up, Down)
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

### "Error 1118: Row size too large"

If using `utf8mb4` with many `VARCHAR` columns, you may hit MySQL's row size limit. Consider:
- Using `TEXT`/`BLOB` for large columns
- Reducing `VARCHAR` lengths
- Using `ROW_FORMAT=DYNAMIC` or `COMPRESSED`