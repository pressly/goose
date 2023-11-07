package goose

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3/database"
)

func (m *Migration) useTx(direction bool) (bool, error) {
	switch m.Type {
	case TypeGo:
		if m.goUp.Mode == 0 || m.goDown.Mode == 0 {
			return false, fmt.Errorf("go migrations must have a mode set")
		}
		if direction {
			return m.goUp.Mode == TransactionEnabled, nil
		}
		return m.goDown.Mode == TransactionEnabled, nil
	case TypeSQL:
		if !m.sql.Parsed {
			return false, fmt.Errorf("sql migrations must be parsed")
		}
		return m.sql.UseTx, nil
	}
	return false, fmt.Errorf("invalid migration type: %q", m.Type)
}

func (m *Migration) isEmpty(direction bool) bool {
	switch m.Type {
	case TypeGo:
		if direction {
			return m.goUp.RunTx == nil && m.goUp.RunDB == nil
		}
		return m.goDown.RunTx == nil && m.goDown.RunDB == nil
	case TypeSQL:
		if direction {
			return len(m.sql.Up) == 0
		}
		return len(m.sql.Down) == 0
	}
	return true
}

func (m *Migration) apply(ctx context.Context, db database.DBTxConn, direction bool) error {
	switch m.Type {
	case TypeGo:
		return runGo(ctx, db, m, direction)
	case TypeSQL:
		if direction {
			return runSQL(ctx, db, m.sql.Up)
		}
		return runSQL(ctx, db, m.sql.Down)
	}
	return fmt.Errorf("invalid migration type: %q", m.Type)
}

func runGo(ctx context.Context, db database.DBTxConn, m *Migration, direction bool) error {
	switch db := db.(type) {
	case *sql.Conn:
		return fmt.Errorf("go migrations are not supported with *sql.Conn")
	case *sql.DB:
		if direction && m.goUp.RunDB != nil {
			return m.goUp.RunDB(ctx, db)
		}
		if !direction && m.goDown.RunDB != nil {
			return m.goDown.RunDB(ctx, db)
		}
		return nil
	case *sql.Tx:
		if direction && m.goUp.RunTx != nil {
			return m.goUp.RunTx(ctx, db)
		}
		if !direction && m.goDown.RunTx != nil {
			return m.goDown.RunTx(ctx, db)
		}
		return nil
	}
	return fmt.Errorf("invalid database connection type: %T", db)
}

func runSQL(ctx context.Context, db database.DBTxConn, statements []string) error {
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
