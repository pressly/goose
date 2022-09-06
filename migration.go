package goose

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// MigrationRecord struct.
type MigrationRecord struct {
	VersionID int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

// GoMigration is a go migration func that is run within a transaction.
type GoMigration func(tx *sql.Tx) error

// GoMigrationNoTx is a go migration funt that is run outside a transaction.
type GoMigrationNoTx func(db *sql.DB) error

// Migration struct.
type Migration struct {
	Version      int64
	Next         int64  // next version, or -1 if none
	Previous     int64  // previous version, -1 if none
	Source       string // path to .sql script or go file
	Registered   bool
	UpFn         GoMigration     // Up go migration function
	DownFn       GoMigration     // Down go migration function
	UpFnNoTx     GoMigrationNoTx // Up go migration function with sql.DB
	DownFnNoTx   GoMigrationNoTx // Down go migration function with sql.DB
	noVersioning bool
}

func (m *Migration) String() string {
	return fmt.Sprintf(m.Source)
}

// Up runs an up migration.
func (m *Migration) Up(db *sql.DB) error {
	if err := m.run(db, true); err != nil {
		return err
	}
	return nil
}

// Down runs a down migration.
func (m *Migration) Down(db *sql.DB) error {
	if err := m.run(db, false); err != nil {
		return err
	}
	return nil
}

func (m *Migration) run(db *sql.DB, direction bool) error {
	switch filepath.Ext(m.Source) {
	case ".sql":
		f, err := baseFS.Open(m.Source)
		if err != nil {
			return fmt.Errorf("ERROR %v: failed to open SQL migration file: %w", filepath.Base(m.Source), err)
		}
		defer f.Close()

		statements, useTx, err := parseSQLMigration(f, direction)
		if err != nil {
			return fmt.Errorf("ERROR %v: failed to parse SQL migration file: %w", filepath.Base(m.Source), err)
		}

		if err := runSQLMigration(db, statements, useTx, m.Version, direction, m.noVersioning); err != nil {
			return fmt.Errorf("ERROR %v: failed to run SQL migration: %w", filepath.Base(m.Source), err)
		}

		if len(statements) > 0 {
			log.Println("OK   ", filepath.Base(m.Source))
		} else {
			log.Println("EMPTY", filepath.Base(m.Source))
		}

	case ".go":
		if !m.Registered {
			return fmt.Errorf("ERROR %v: failed to run Go migration: Go functions must be registered and built into a custom binary (see https://github.com/pressly/goose/tree/master/examples/go-migrations)", m.Source)
		}
		fn := m.UpFn
		fnNoTx := m.UpFnNoTx
		if !direction {
			fn = m.DownFn
			fnNoTx = m.DownFnNoTx
		}

		if fnNoTx != nil {
			// Run Go migration function with *sql.DB
			if err := fnNoTx(db); err != nil {
				return fmt.Errorf("ERROR %v: failed to run Go migration function %T: %w", filepath.Base(m.Source), fnNoTx, err)
			}
			if m.noVersioning {
				return nil
			}
			if err := insertOrDeleteVersionNoTx(db, direction, m.Version); err != nil {
				return fmt.Errorf("ERROR %v: failed to insert version with no tx: %w", filepath.Base(m.Source), err)
			}
			return nil
		}

		if fn != nil {
			// Run Go migration function.
			tx, err := db.Begin()
			if err != nil {
				return fmt.Errorf("ERROR failed to begin transaction: %w", err)
			}
			if err := fn(tx); err != nil {
				if outerErr := tx.Rollback(); outerErr != nil {
					return fmt.Errorf("ERROR %v: failed to run go migration: rollback error: %s: %w",
						filepath.Base(m.Source),
						err,
						outerErr,
					)
				}
				return fmt.Errorf("ERROR %v: failed to run Go migration function %T: %w", filepath.Base(m.Source), fn, err)
			}
			if !m.noVersioning {
				if err := insertOrDeleteVersion(tx, direction, m.Version); err != nil {
					if outerErr := tx.Rollback(); outerErr != nil {
						return fmt.Errorf("ERROR %v: failed to insert version: rollback error: %s: %w",
							filepath.Base(m.Source),
							err,
							outerErr,
						)
					}
					return err
				}
			}
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("ERROR %v: failed to run go migration: commit error : %T: %w", filepath.Base(m.Source), fn, err)
			}
			return nil
		}

		if fn != nil || fnNoTx != nil {
			log.Println("OK   ", filepath.Base(m.Source))
		} else {
			log.Println("EMPTY", filepath.Base(m.Source))
		}

		return nil
	}

	return nil
}

func insertOrDeleteVersionNoTx(db *sql.DB, direction bool, version int64) error {
	if direction {
		_, err := db.Exec(GetDialect().insertVersionSQL(), version, direction)
		return err
	}
	_, err := db.Exec(GetDialect().deleteVersionSQL(), version)
	return err
}

func insertOrDeleteVersion(tx *sql.Tx, direction bool, version int64) error {
	if direction {
		_, err := tx.Exec(GetDialect().insertVersionSQL(), version, direction)
		return err
	}
	_, err := tx.Exec(GetDialect().deleteVersionSQL(), version)
	return err
}

// NumericComponent looks for migration scripts with names in the form:
// XXX_descriptivename.ext where XXX specifies the version number
// and ext specifies the type of migration
func NumericComponent(name string) (int64, error) {
	base := filepath.Base(name)

	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("not a recognized migration file type")
	}

	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no filename separator '_' found")
	}

	n, e := strconv.ParseInt(base[:idx], 10, 64)
	if e == nil && n <= 0 {
		return 0, errors.New("migration IDs must be greater than zero")
	}

	return n, e
}
