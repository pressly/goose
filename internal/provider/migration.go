package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/pressly/goose/v3/internal/sqlextended"
)

type migration struct {
	Source Source
	// A migration is either a Go migration or a SQL migration, but never both.
	//
	// Note, the SQLParsed field is used to determine if the SQL migration has been parsed. This is
	// an optimization to avoid parsing the SQL migration if it is never required. Also, the
	// majority of the time migrations are incremental, so it is likely that the user will only want
	// to run the last few migrations, and there is no need to parse ALL prior migrations.
	//
	// Exactly one of these fields will be set:
	Go *goMigration
	// -- OR --
	SQL *sqlMigration
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
		return fmt.Sprintf("unknown (%d)", t)
	}
}

func (m *migration) GetSQLStatements(direction bool) ([]string, error) {
	if m.Source.Type != TypeSQL {
		return nil, fmt.Errorf("expected sql migration, got %s: no sql statements", m.Source.Type)
	}
	if m.SQL == nil {
		return nil, errors.New("sql migration has not been parsed")
	}
	if direction {
		return m.SQL.UpStatements, nil
	}
	return m.SQL.DownStatements, nil
}

func (g *goMigration) run(ctx context.Context, tx *sql.Tx, direction bool) error {
	if g == nil {
		return nil
	}
	var fn func(context.Context, *sql.Tx) error
	if direction && g.up != nil {
		fn = g.up.Run
	}
	if !direction && g.down != nil {
		fn = g.down.Run
	}
	if fn != nil {
		return fn(ctx, tx)
	}
	return nil
}

func (g *goMigration) runNoTx(ctx context.Context, db *sql.DB, direction bool) error {
	if g == nil {
		return nil
	}
	var fn func(context.Context, *sql.DB) error
	if direction && g.up != nil {
		fn = g.up.RunNoTx
	}
	if !direction && g.down != nil {
		fn = g.down.RunNoTx
	}
	if fn != nil {
		return fn(ctx, db)
	}
	return nil
}

type sqlMigration struct {
	UseTx          bool
	UpStatements   []string
	DownStatements []string
}

func (s *sqlMigration) IsEmpty(direction bool) bool {
	if direction {
		return len(s.UpStatements) == 0
	}
	return len(s.DownStatements) == 0
}

func (s *sqlMigration) run(ctx context.Context, db sqlextended.DBTxConn, direction bool) error {
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
