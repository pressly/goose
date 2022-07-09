package e2e

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
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
	ctx := context.Background()
	db, err := newDockerDB(t)
	check.NoError(t, err)
	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)

	err = p.Up(ctx)
	check.NoError(t, err)
	baseVersion, err := p.GetDBVersion(ctx)
	check.NoError(t, err)

	// Create a separate provider
	options := &goose.Options{
		NoVersioning: true,
	}
	noVersionProvider, err := goose.NewProvider(toDialect(t, *dialect), db, seedDir, options)
	check.NoError(t, err)

	t.Run("seed-up-down-to-zero", func(t *testing.T) {
		// Run (all) up migrations from the seed dir
		{
			err = noVersionProvider.Up(ctx)
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			dbVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, dbVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}

		// Run (all) down migrations from the seed dir
		{
			err = noVersionProvider.DownTo(ctx, 0)
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			dbVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, dbVersion)
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
			err = noVersionProvider.Up(ctx)
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			dbVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, dbVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}

		// Run reset (effectively the same as down-to 0)
		{
			err = noVersionProvider.Reset(ctx)
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			dbVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, dbVersion)
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
			err = noVersionProvider.Up(ctx)
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			dbVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, dbVersion)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}

		// Run redo (effectively the same as down and up by one)
		{
			err = noVersionProvider.Redo(ctx)
			check.NoError(t, err)
			// Confirm no changes to the versioned schema in the DB
			dbVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, dbVersion)
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
