package e2e

import (
	"database/sql"
	"testing"

	"github.com/matryer/is"
	"github.com/pressly/goose/v3"
)

func TestUpNoVersioning(t *testing.T) {
	if *dialect != dialectPostgres {
		t.SkipNow()
	}
	const (
		wantSeedOwnerCount = 250
		// These are owners created by migration files.
		wantOwnerCount = 4
	)
	is := is.New(t)
	db, err := newDockerDB(t)
	is.NoErr(err)
	goose.SetDialect(*dialect)

	err = goose.Up(db, migrationsDir)
	is.NoErr(err)
	baseVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)

	// Run (all) up migrations from the seed dir
	{
		err = goose.Up(db, seedDir, goose.WithNoVersioning())
		is.NoErr(err)
		// Confirm no changes to the versioned schema in the DB
		currentVersion, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(baseVersion, currentVersion)
		seedOwnerCount, err := countSeedOwners(db)
		is.NoErr(err)
		is.Equal(seedOwnerCount, wantSeedOwnerCount)
	}

	// Run (all) down migrations from the seed dir
	{
		err = goose.DownTo(db, seedDir, 0, goose.WithNoVersioning())
		is.NoErr(err)
		// Confirm no changes to the versioned schema in the DB
		currentVersion, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(baseVersion, currentVersion)
		seedOwnerCount, err := countSeedOwners(db)
		is.NoErr(err)
		is.Equal(seedOwnerCount, 0)
	}

	// The migrations added 4 non-seed owners, they must remain
	// in the database afterwards
	ownerCount, err := countOwners(db)
	is.NoErr(err)
	is.Equal(ownerCount, wantOwnerCount)
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
