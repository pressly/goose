
package migration_003

import (
    "database/sql"
    "fmt"
)

func Up(txn *sql.Tx) {
    fmt.Println("Hello from migration_003 Up!")
}

func Down(txn *sql.Tx) {
    fmt.Println("Hello from migration_003 Down!")
}
