package migrate

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3/internal/sqlextended"
)

type Migration struct {
	// Fullpath is the full path to the migration file.
	//
	// Example: /path/to/migrations/123_create_users_table.go
	Fullpath string
	// Version is the version of the migration.
	Version int64
	// Type is the type of migration.
	Type MigrationType
	// A migration is either a Go migration or a SQL migration, but never both.
	//
	// Note, the SQLParsed field is used to determine if the SQL migration has been parsed. This is
	// an optimization to avoid parsing the SQL migration if it is never required. Also, the
	// majority of the time migrations are incremental, so it is likely that the user will only want
	// to run the last few migrations, and there is no need to parse ALL prior migrations.
	//
	// Exactly one of these fields will be set:
	Go *Go
	// -- or --
	SQLParsed bool
	SQL       *SQL
}

type MigrationType int

const (
	TypeGo MigrationType = iota + 1
	TypeSQL
)

func (t MigrationType) String() string {
	switch t {
	case TypeGo:
		return "go"
	case TypeSQL:
		return "sql"
	default:
		// This should never happen.
		return "unknown"
	}
}

func (m *Migration) UseTx() bool {
	switch m.Type {
	case TypeGo:
		return m.Go.UseTx
	case TypeSQL:
		return m.SQL.UseTx
	default:
		// This should never happen.
		panic("unknown migration type: use tx")
	}
}

func (m *Migration) IsEmpty(direction bool) bool {
	switch m.Type {
	case TypeGo:
		return m.Go.IsEmpty(direction)
	case TypeSQL:
		return m.SQL.IsEmpty(direction)
	default:
		// This should never happen.
		panic("unknown migration type: is empty")
	}
}

func (m *Migration) GetSQLStatements(direction bool) ([]string, error) {
	if m.Type != TypeSQL {
		return nil, fmt.Errorf("expected sql migration, got %s: no sql statements", m.Type)
	}
	if m.SQL == nil {
		return nil, errors.New("sql migration has not been initialized")
	}
	if !m.SQLParsed {
		return nil, errors.New("sql migration has not been parsed")
	}
	if direction {
		return m.SQL.UpStatements, nil
	}
	return m.SQL.DownStatements, nil
}

type Go struct {
	// We used an explicit bool instead of relying on a pointer because registered funcs may be nil.
	// These are still valid Go and versioned migrations, but they are just empty.
	//
	// For example: goose.AddMigration(nil, nil)
	UseTx bool

	// Only one of these func pairs will be set:
	UpFn, DownFn func(context.Context, *sql.Tx) error
	// -- or --
	UpFnNoTx, DownFnNoTx func(context.Context, *sql.DB) error
}

func (g *Go) IsEmpty(direction bool) bool {
	if direction {
		return g.UpFn == nil && g.UpFnNoTx == nil
	}
	return g.DownFn == nil && g.DownFnNoTx == nil
}

func (g *Go) run(ctx context.Context, tx *sql.Tx, direction bool) error {
	var fn func(context.Context, *sql.Tx) error
	if direction {
		fn = g.UpFn
	} else {
		fn = g.DownFn
	}
	if fn != nil {
		return fn(ctx, tx)
	}
	return nil
}

func (g *Go) runNoTx(ctx context.Context, db *sql.DB, direction bool) error {
	var fn func(context.Context, *sql.DB) error
	if direction {
		fn = g.UpFnNoTx
	} else {
		fn = g.DownFnNoTx
	}
	if fn != nil {
		return fn(ctx, db)
	}
	return nil
}

type SQL struct {
	UseTx          bool
	UpStatements   []string
	DownStatements []string
}

func (s *SQL) IsEmpty(direction bool) bool {
	if direction {
		return len(s.UpStatements) == 0
	}
	return len(s.DownStatements) == 0
}

func (s *SQL) run(ctx context.Context, db sqlextended.DBTxConn, direction bool) error {
	var statements []string
	if direction {
		statements = s.UpStatements
	} else {
		statements = s.DownStatements
	}
	for _, stmt := range statements {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}
