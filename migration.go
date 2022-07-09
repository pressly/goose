package goose

import (
	"fmt"
	"time"
)

// MigrationRecord struct.
type MigrationRecord struct {
	VersionID int64
	TStamp    time.Time
	IsApplied bool // was this a result of up() or down()
}

// Migration struct.
type Migration struct {
	Version  int64
	Next     int64  // next version, or -1 if none
	Previous int64  // previous version, -1 if none
	Source   string // path to .sql script or go file
	// With txn
	Registered bool
	UpFn       GoMigration
	DownFn     GoMigration
	// Without tx
	UpFnNoTx   GoMigrationNoTx
	DownFnNoTx GoMigrationNoTx

	noVersioning bool
}

func (m *Migration) String() string {
	return fmt.Sprintf(m.Source)
}
