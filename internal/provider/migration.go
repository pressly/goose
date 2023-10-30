package provider

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/pressly/goose/v3/database"
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

func (m *migration) useTx(direction bool) bool {
	switch m.Source.Type {
	case TypeSQL:
		return m.SQL.UseTx
	case TypeGo:
		if m.Go == nil || m.Go.isEmpty(direction) {
			return false
		}
		if direction {
			return m.Go.up.Run != nil
		}
		return m.Go.down.Run != nil
	}
	// This should never happen.
	return false
}

func (m *migration) isEmpty(direction bool) bool {
	switch m.Source.Type {
	case TypeSQL:
		return m.SQL == nil || m.SQL.isEmpty(direction)
	case TypeGo:
		return m.Go == nil || m.Go.isEmpty(direction)
	}
	return true
}

func (m *migration) filename() string {
	return filepath.Base(m.Source.Path)
}

// run runs the migration inside of a transaction.
func (m *migration) run(ctx context.Context, tx *sql.Tx, direction bool) error {
	switch m.Source.Type {
	case TypeSQL:
		if m.SQL == nil {
			return fmt.Errorf("tx: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, tx, direction)
	case TypeGo:
		return m.Go.run(ctx, tx, direction)
	}
	// This should never happen.
	return fmt.Errorf("tx: failed to run migration %s: neither sql or go", filepath.Base(m.Source.Path))
}

// runNoTx runs the migration without a transaction.
func (m *migration) runNoTx(ctx context.Context, db *sql.DB, direction bool) error {
	switch m.Source.Type {
	case TypeSQL:
		if m.SQL == nil {
			return fmt.Errorf("db: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, db, direction)
	case TypeGo:
		return m.Go.runNoTx(ctx, db, direction)
	}
	// This should never happen.
	return fmt.Errorf("db: failed to run migration %s: neither sql or go", filepath.Base(m.Source.Path))
}

// runConn runs the migration without a transaction using the provided connection.
func (m *migration) runConn(ctx context.Context, conn *sql.Conn, direction bool) error {
	switch m.Source.Type {
	case TypeSQL:
		if m.SQL == nil {
			return fmt.Errorf("conn: sql migration has not been parsed")
		}
		return m.SQL.run(ctx, conn, direction)
	case TypeGo:
		return fmt.Errorf("conn: go migrations are not supported with *sql.Conn")
	}
	// This should never happen.
	return fmt.Errorf("conn: failed to run migration %s: neither sql or go", filepath.Base(m.Source.Path))
}

type goMigration struct {
	fullpath string
	up, down *GoMigrationFunc
}

func (g *goMigration) isEmpty(direction bool) bool {
	if g.up == nil && g.down == nil {
		panic("go migration has no up or down")
	}
	if direction {
		return g.up == nil
	}
	return g.down == nil
}

func newGoMigration(fullpath string, up, down *GoMigrationFunc) *goMigration {
	return &goMigration{
		fullpath: fullpath,
		up:       up,
		down:     down,
	}
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

func (s *sqlMigration) isEmpty(direction bool) bool {
	if direction {
		return len(s.UpStatements) == 0
	}
	return len(s.DownStatements) == 0
}

func (s *sqlMigration) run(ctx context.Context, db database.DBTxConn, direction bool) error {
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
