package migration

import (
	"context"
	"database/sql"
	"errors"
)

type MigrationType int

const (
	TypeGo MigrationType = iota + 1
	TypeSQL
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
	// an optimization to avoid parsing the SQL migration if it is never required. Also, the majority
	// of the time migrations are incremental, so it is likely that the user will only want to run
	// the last few migrations, and there is no need to parse ALL prior migrations.
	//
	// Only one of these fields will be set:
	Go *Go
	// -- or --
	SQLParsed bool
	SQL       *SQL
}

func (m *Migration) IsGo() bool {
	return m.Type == TypeGo
}

func (m *Migration) IsSQL() bool {
	return m.Type == TypeSQL
}

func (m *Migration) UseTx() bool {
	if m.IsGo() {
		return m.Go.UseTx
	}
	if m.IsSQL() {
		return m.SQL.UseTx
	}
	return false
}

func (m *Migration) GetSQLStatements(direction bool) ([]string, error) {
	if m.IsGo() {
		return nil, errors.New("go migration does not have sql statements")
	}
	if !m.SQLParsed || m.SQL == nil {
		return nil, errors.New("sql migration has not been parsed")
	}
	if direction {
		return m.SQL.UpStatements, nil
	}
	return m.SQL.DownStatements, nil
}

func (m *Migration) IsEmpty(direction bool) bool {
	if m.IsGo() {
		return m.Go.IsEmpty(direction)
	}
	if m.IsSQL() {
		return m.SQL.IsEmpty(direction)
	}
	return false
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
	fn := g.DownFn
	if direction {
		fn = g.UpFn
	}
	if fn != nil {
		return fn(ctx, tx)
	}
	return nil
}

func (g *Go) runNoTx(ctx context.Context, db *sql.DB, direction bool) error {
	fn := g.DownFnNoTx
	if direction {
		fn = g.UpFnNoTx
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

// DBTxConn is an interface that is satisfied by *sql.DB, *sql.Tx and *sql.Conn.
//
// There is a long outstanding issue to formalize a std lib interface, but alas...
// See: https://github.com/golang/go/issues/14468
type DBTxConn interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

func (s *SQL) run(ctx context.Context, db DBTxConn, direction bool) error {
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
