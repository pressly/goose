package database

import (
	"database/sql"

	"github.com/pressly/goose/v3/database/dialect"
)

// DBTxConn is a thin interface for common methods that is satisfied by *sql.DB, *sql.Tx and
// *sql.Conn.
//
// There is a long outstanding issue to formalize a std lib interface, but alas. See:
// https://github.com/golang/go/issues/14468
type DBTxConn = dialect.DBTxConn

var (
	_ DBTxConn = (*sql.DB)(nil)
	_ DBTxConn = (*sql.Tx)(nil)
	_ DBTxConn = (*sql.Conn)(nil)
)

