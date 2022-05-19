package e2e

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestNoVersioning(t *testing.T) {
	if *dialect != dialectPostgres {
		t.SkipNow()
	}
	const (
		// Total owners created by the seed files.
		wantSeedOwnerCount = 250
		// These are owners created by migration files.
		wantOwnerCount = 4
	)
	db, err := newDockerDB(t)
	check.NoError(t, err)
	goose.SetDialect(*dialect)

	err = goose.Up(db, migrationsDir)
	check.NoError(t, err)
	baseVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)

	t.Run("seed-up-down-to-zero", func(t *testing.T) {
		// Run (all) up migrations from the seed dir
		{
			err = goose.Up(db, seedDir, goose.WithNoVersioning())
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := goose.GetDBVersion(db)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}

		// Run (all) down migrations from the seed dir
		{
			err = goose.DownTo(db, seedDir, 0, goose.WithNoVersioning())
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := goose.GetDBVersion(db)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, 0)
		}

		// The migrations added 4 non-seed owners, they must remain
		// in the database afterwards
		ownerCount, err := countOwners(db)
		check.NoError(t, err)
		check.Number(t, ownerCount, wantOwnerCount)
	})

	t.Run("test-seed-up-reset", func(t *testing.T) {
		// Run (all) up migrations from the seed dir
		{
			err = goose.Up(db, seedDir, goose.WithNoVersioning())
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := goose.GetDBVersion(db)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}

		// Run reset (effectively the same as down-to 0)
		{
			err = goose.Reset(db, seedDir, goose.WithNoVersioning())
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := goose.GetDBVersion(db)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, 0)
		}

		// The migrations added 4 non-seed owners, they must remain
		// in the database afterwards
		ownerCount, err := countOwners(db)
		check.NoError(t, err)
		check.Number(t, ownerCount, wantOwnerCount)
	})

	t.Run("test-seed-up-redo", func(t *testing.T) {
		// Run (all) up migrations from the seed dir
		{
			err = goose.Up(db, seedDir, goose.WithNoVersioning())
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := goose.GetDBVersion(db)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}

		// Run reset (effectively the same as down-to 0)
		{
			err = goose.Redo(db, seedDir, goose.WithNoVersioning())
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := goose.GetDBVersion(db)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount) // owners should be unchanged
		}

		// The migrations added 4 non-seed owners, they must remain
		// in the database afterwards along with the 250 seed owners for a
		// total of 254.
		ownerCount, err := countOwners(db)
		check.NoError(t, err)
		check.Number(t, ownerCount, wantOwnerCount+wantSeedOwnerCount)
	})
}

func countSeedOwners(db *sql.DB) (int, error) {
	q := `SELECT count(*)FROM owners WHERE owner_name LIKE'seed-user-%'`
	var count int
	if err := db.QueryRow(q).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func countOwners(db *sql.DB) (int, error) {
	q := `SELECT count(*)FROM owners`
	var count int
	if err := db.QueryRow(q).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
