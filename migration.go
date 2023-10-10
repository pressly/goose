package goose

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pressly/goose/v3/internal/sqlparser"
	"github.com/pressly/goose/v3/state"
)

// MigrationRecord struct.
type MigrationRecord struct {
	VersionID int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

// Migration struct.
type Migration struct {
	Version    int64
	Next       int64  // next version, or -1 if none
	Previous   int64  // previous version, -1 if none
	Source     string // path to .sql script or go file
	Registered bool
	UseTx      bool

	// These are deprecated and will be removed in the future.
	// For backwards compatibility we still save the non-context versions in the struct in case someone is using them.
	// Goose does not use these internally anymore and instead uses the context versions.
	UpFn, DownFn         GoMigration
	UpFnNoTx, DownFnNoTx GoMigrationNoTx

	// New functions with context
	UpFnContext, DownFnContext         GoMigrationContext
	UpFnNoTxContext, DownFnNoTxContext GoMigrationNoTxContext
	noVersioning                       bool
}

func (m *Migration) String() string {
	return fmt.Sprint(m.Source)
}

// Up runs an up migration.
func (m *Migration) Up(db *sql.DB) error {
	ctx := context.Background()
	return m.UpContext(ctx, db)
}

// UpContext runs an up migration.
func (m *Migration) UpContext(ctx context.Context, db *sql.DB) error {
	if err := m.run(ctx, db, true); err != nil {
		return err
	}
	return nil
}

// Down runs a down migration.
func (m *Migration) Down(db *sql.DB) error {
	ctx := context.Background()
	return m.DownContext(ctx, db)
}

// DownContext runs a down migration.
func (m *Migration) DownContext(ctx context.Context, db *sql.DB) error {
	if err := m.run(ctx, db, false); err != nil {
		return err
	}
	return nil
}

func (m *Migration) run(ctx context.Context, db *sql.DB, direction bool) error {
	switch filepath.Ext(m.Source) {
	case ".sql":
		f, err := baseFS.Open(m.Source)
		if err != nil {
			return fmt.Errorf("ERROR %v: failed to open SQL migration file: %w", filepath.Base(m.Source), err)
		}
		defer f.Close()

		statements, useTx, err := sqlparser.ParseSQLMigration(f, sqlparser.FromBool(direction), verbose)
		if err != nil {
			return fmt.Errorf("ERROR %v: failed to parse SQL migration file: %w", filepath.Base(m.Source), err)
		}

		start := time.Now()
		if err := runSQLMigration(ctx, db, statements, useTx, m.Version, direction, m.noVersioning); err != nil {
			return fmt.Errorf("ERROR %v: failed to run SQL migration: %w", filepath.Base(m.Source), err)
		}
		finish := truncateDuration(time.Since(start))

		if len(statements) > 0 {
			log.Printf("OK   %s (%s)\n", filepath.Base(m.Source), finish)
		} else {
			log.Printf("EMPTY %s (%s)\n", filepath.Base(m.Source), finish)
		}

	case ".go":
		if !m.Registered {
			return fmt.Errorf("ERROR %v: failed to run Go migration: Go functions must be registered and built into a custom binary (see https://github.com/pressly/goose/tree/master/examples/go-migrations)", m.Source)
		}
		start := time.Now()
		var empty bool
		if m.UseTx {
			// Run go-based migration inside a tx.
			fn := m.DownFnContext
			if direction {
				fn = m.UpFnContext
			}
			empty = (fn == nil)
			if err := runGoMigration(
				ctx,
				db,
				fn,
				m.Version,
				direction,
				!m.noVersioning,
			); err != nil {
				return fmt.Errorf("ERROR go migration: %q: %w", filepath.Base(m.Source), err)
			}
		} else {
			// Run go-based migration outside a tx.
			fn := m.DownFnNoTxContext
			if direction {
				fn = m.UpFnNoTxContext
			}
			empty = (fn == nil)
			if err := runGoMigrationNoTx(
				ctx,
				db,
				fn,
				m.Version,
				direction,
				!m.noVersioning,
			); err != nil {
				return fmt.Errorf("ERROR go migration no tx: %q: %w", filepath.Base(m.Source), err)
			}
		}
		finish := truncateDuration(time.Since(start))
		if !empty {
			log.Printf("OK   %s (%s)\n", filepath.Base(m.Source), finish)
		} else {
			log.Printf("EMPTY %s (%s)\n", filepath.Base(m.Source), finish)
		}
	}
	return nil
}

func runGoMigrationNoTx(
	ctx context.Context,
	db *sql.DB,
	fn GoMigrationNoTxContext,
	version int64,
	direction bool,
	recordVersion bool,
) error {
	if fn != nil {
		// Run go migration function.
		if err := fn(ctx, db); err != nil {
			return fmt.Errorf("failed to run go migration: %w", err)
		}
	}
	if recordVersion {
		return insertOrDeleteVersion(ctx, db, version, direction)
	}
	return nil
}

func runGoMigration(
	ctx context.Context,
	db *sql.DB,
	fn GoMigrationContext,
	version int64,
	direction bool,
	recordVersion bool,
) error {
	if fn == nil && !recordVersion {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	if fn != nil {
		// Run go migration function.
		if err := fn(ctx, tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to run go migration: %w", err)
		}
	}
	if recordVersion {
		if err := insertOrDeleteVersion(ctx, tx, version, direction); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("failed to update version: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

func insertOrDeleteVersion(ctx context.Context, db state.DB, version int64, direction bool) error {
	if direction {
		return globalStorage().InsertVersion(ctx, db, version)
	}
	return globalStorage().DeleteVersion(ctx, db, version)
}

// NumericComponent parses the version from the migration file name.
//
// XXX_descriptivename.ext where XXX specifies the version number and ext specifies the type of
// migration, either .sql or .go.
func NumericComponent(filename string) (int64, error) {
	base := filepath.Base(filename)
	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("migration file does not have .sql or .go file extension")
	}
	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no filename separator '_' found")
	}
	n, err := strconv.ParseInt(base[:idx], 10, 64)
	if err != nil {
		return 0, err
	}
	if n < 1 {
		return 0, errors.New("migration version must be greater than zero")
	}
	return n, nil
}

func truncateDuration(d time.Duration) time.Duration {
	for _, v := range []time.Duration{
		time.Second,
		time.Millisecond,
		time.Microsecond,
	} {
		if d > v {
			return d.Round(v / time.Duration(100))
		}
	}
	return d
}
