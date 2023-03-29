package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
)

func TestNoVersioning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	const (
		// Total owners created by the seed files.
		wantSeedOwnerCount = 250
		// These are owners created by migration files.
		wantOwnerCount = 4
	)
	te := newTestEnv(t, migrationsDir, nil)
	check.NumberNotZero(t, len(te.provider.ListMigrations()))

	upResult, err := te.provider.Up(ctx)
	check.NoError(t, err)
	check.Number(t, len(upResult), 11)
	baseVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, baseVersion, 11)

	t.Run("seed-up-down-to-zero", func(t *testing.T) {
		options := goose.DefaultOptions().
			SetVerbose(testing.Verbose()).
			SetDir(seedDir).
			SetNoVersioning(true)
		p, err := goose.NewProvider(goose.DialectPostgres, te.db, options)
		check.NoError(t, err)
		check.Number(t, len(p.ListMigrations()), 2)

		// Run (all) up migrations from the seed dir
		{
			upResult, err := p.Up(ctx)
			check.NoError(t, err)
			check.Number(t, len(upResult), 2)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(te.db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}
		// Run (all) down migrations from the seed dir
		{
			downResult, err := p.DownTo(ctx, 0)
			check.NoError(t, err)
			check.Number(t, len(downResult), 2)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(te.db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, 0)
		}
		// The migrations added 4 non-seed owners, they must remain
		// in the database afterwards
		ownerCount, err := countOwners(te.db)
		check.NoError(t, err)
		check.Number(t, ownerCount, wantOwnerCount)
	})

	t.Run("test-seed-up-reset", func(t *testing.T) {
		options := goose.DefaultOptions().
			SetVerbose(testing.Verbose()).
			SetDir(seedDir).
			SetNoVersioning(true)
		p, err := goose.NewProvider(goose.DialectPostgres, te.db, options)
		check.NoError(t, err)

		// Run (all) up migrations from the seed dir
		{
			upResult, err = p.Up(ctx)
			check.NoError(t, err)
			check.Number(t, len(upResult), 2)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(te.db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}
		// Run reset (effectively the same as down-to 0)
		{
			resetResult, err := p.Reset(ctx)
			check.NoError(t, err)
			check.Number(t, len(resetResult), 2)
			// Confirm no changes to the versioned schema in the DB
			currentVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, baseVersion, currentVersion)
			seedOwnerCount, err := countSeedOwners(te.db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, 0)
		}
		// The migrations added 4 non-seed owners, they must remain
		// in the database afterwards
		ownerCount, err := countOwners(te.db)
		check.NoError(t, err)
		check.Number(t, ownerCount, wantOwnerCount)
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
