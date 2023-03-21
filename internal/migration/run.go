package migration

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
)

func (m *Migration) Run(ctx context.Context, tx *sql.Tx, direction bool) error {
	switch {
	case m.IsSQL():
		if !m.SQLParsed || m.SQL == nil {
			return fmt.Errorf("tx: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, tx, direction)
	case m.IsGo():
		return m.Go.run(ctx, tx, direction)
	}
	// This should never happen.
	return fmt.Errorf("tx: failed to run migration (%s): neither sql or go", filepath.Base(m.Fullpath))
}

func (m *Migration) RunNoTx(ctx context.Context, db *sql.DB, direction bool) error {
	switch {
	case m.IsSQL():
		if !m.SQLParsed || m.SQL == nil {
			return fmt.Errorf("db: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, db, direction)
	case m.IsGo():
		return m.Go.runNoTx(ctx, db, direction)
	}
	// This should never happen.
	return fmt.Errorf("db: failed to run migration (%s): neither sql or go", filepath.Base(m.Fullpath))
}

func (m *Migration) RunConn(ctx context.Context, conn *sql.Conn, direction bool) error {
	switch {
	case m.IsSQL():
		if !m.SQLParsed || m.SQL == nil {
			return fmt.Errorf("conn: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, conn, direction)
	case m.IsGo():
		return fmt.Errorf("conn: go migrations are not supported with *sql.Conn")
	}
	// This should never happen.
	return fmt.Errorf("failed to run migration (%s): neither sql or go", filepath.Base(m.Fullpath))
}
