package goose

import (
	"database/sql"
	"fmt"
	"runtime"
)

// GoMigration is a Go migration func that is run within a transaction.
type GoMigration func(tx *sql.Tx) error

// GoMigrationNoTx is a Go migration func that is run outside a transaction.
type GoMigrationNoTx func(db *sql.DB) error

// AddMigration adds a Go migration.
//
// This function is intended to be called from a versioned Go migration file, and will
// panic at build time if a duplicate version is detected.
//
// Example:
//
//	func init() {
//		goose.AddMigration(Up00002, Down00002)
//	}
func AddMigration(up, down GoMigration) {
	_, filename, _, _ := runtime.Caller(1)
	if err := register(filename, true, up, down, nil, nil); err != nil {
		panic(err)
	}
}

// AddMigrationNoTx adds a Go migration that will be run outside a transaction.
//
// This function is intended to be called from a versioned Go migration file, and will
// panic at build time if a duplicate version is detected.
//
// Example:
//
//	func init() {
//		goose.AddMigrationNoTx(Up00002, Down00002)
//	}
func AddMigrationNoTx(up, down GoMigrationNoTx) {
	_, filename, _, _ := runtime.Caller(1)
	if err := register(filename, false, nil, nil, up, down); err != nil {
		panic(err)
	}
}

func register(
	filename string,
	useTx bool,
	up, down GoMigration,
	upNoTx, downNoTx GoMigrationNoTx,
) error {
	// Sanity check caller did not mix tx and non-tx based functions.
	if (up != nil || down != nil) && (upNoTx != nil || downNoTx != nil) {
		return fmt.Errorf("cannot mix tx and non-tx based Go migration functions")
	}
	version, _ := NumericComponent(filename)
	if existing, ok := registeredGoMigrations[version]; ok {
		return fmt.Errorf("failed to add migration %q: version %d conflicts with %q",
			filename,
			version,
			existing.source,
		)
	}
	// Add to global as a registered migration.
	registeredGoMigrations[version] = &migration{
		source:        filename,
		version:       version,
		migrationType: MigrationTypeGo,
		goMigration: &goMigration{
			useTx:      useTx,
			upFn:       up,
			downFn:     down,
			upFnNoTx:   upNoTx,
			downFnNoTx: downNoTx,
		},
	}
	return nil
}
