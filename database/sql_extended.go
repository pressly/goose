package database

import (
	"github.com/pressly/goose/v3/internal/sql"
)

// DBTxConn is a thin interface for common methods that is satisfied by *sql.DB, *sql.Tx and
// *sql.Conn.
//
// There is a long outstanding issue to formalize a std lib interface, but alas. See:
// https://github.com/golang/go/issues/14468
type DBTxConn = sql.DBTxConn
