package goose

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/pressly/goose/v3/database"
)

// useTx is a helper function that returns true if the migration should be run in a transaction. It
// must only be called after the migration has been parsed and initialized.
func useTx(m *Migration, direction bool) (bool, error) {
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
	return false, fmt.Errorf("use tx: invalid migration type: %q", m.Type)
}

// isEmpty is a helper function that returns true if the migration has no functions or no statements
// to execute. It must only be called after the migration has been parsed and initialized.
func isEmpty(m *Migration, direction bool) bool {
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

// runMigration is a helper function that runs the migration in the given direction. It must only be
// called after the migration has been parsed and initialized.
func runMigration(ctx context.Context, db database.DBTxConn, m *Migration, direction bool) error {
	switch m.Type {
	case TypeGo:
		return runGo(ctx, db, m, direction)
	case TypeSQL:
		return runSQL(ctx, db, m, direction)
	}
	return fmt.Errorf("invalid migration type: %q", m.Type)
}

// runGo is a helper function that runs the given Go functions in the given direction. It must only
// be called after the migration has been initialized.
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

// runSQL is a helper function that runs the given SQL statements in the given direction. It must
// only be called after the migration has been parsed.
func runSQL(ctx context.Context, db database.DBTxConn, m *Migration, direction bool) error {
	if !m.sql.Parsed {
		return fmt.Errorf("sql migrations must be parsed")
	}
	var statements []string
	if direction {
		statements = m.sql.Up
	} else {
		statements = m.sql.Down
	}
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
