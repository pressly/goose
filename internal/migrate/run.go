package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
)

// Run runs the migration inside of a transaction.
func (m *Migration) Run(ctx context.Context, tx *sql.Tx, direction bool) error {
	switch m.Type {
	case TypeSQL:
		if m.SQL == nil || !m.SQLParsed {
			return fmt.Errorf("tx: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, tx, direction)
	case TypeGo:
		return m.Go.run(ctx, tx, direction)
	}
	// This should never happen.
	return fmt.Errorf("tx: failed to run migration %s: neither sql or go", filepath.Base(m.Fullpath))
}

// RunNoTx runs the migration without a transaction.
func (m *Migration) RunNoTx(ctx context.Context, db *sql.DB, direction bool) error {
	switch m.Type {
	case TypeSQL:
		if m.SQL == nil || !m.SQLParsed {
			return fmt.Errorf("db: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, db, direction)
	case TypeGo:
		return m.Go.runNoTx(ctx, db, direction)
	}
	// This should never happen.
	return fmt.Errorf("db: failed to run migration %s: neither sql or go", filepath.Base(m.Fullpath))
}

// RunConn runs the migration without a transaction using the provided connection.
func (m *Migration) RunConn(ctx context.Context, conn *sql.Conn, direction bool) error {
	switch m.Type {
	case TypeSQL:
		if m.SQL == nil || !m.SQLParsed {
			return fmt.Errorf("conn: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, conn, direction)
	case TypeGo:
		return fmt.Errorf("conn: go migrations are not supported with *sql.Conn")
	}
	// This should never happen.
	return fmt.Errorf("conn: failed to run migration %s: neither sql or go", filepath.Base(m.Fullpath))
}
