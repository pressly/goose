package e2e

import (
	"database/sql"
	"strings"
	"testing"

	"github.com/matryer/is"
	"github.com/pressly/goose/v3"
)

func TestNotAllowMissing(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// Create and apply first 5 migrations.
	db := setupAllowMissingTestDB(t, 5)

	// Developer A and B check out the "main" branch which is currently
	// on version 5. Developer A mistakenly creates migration 7 and commits.
	// Developer B did not pull the latest changes and commits migration 6. Oops.

	// Developer A - migration 7 (mistakenly applied)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, 7)
	is.NoErr(err)
	err = migrations[6].Up(db)
	is.NoErr(err)
	current, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(current, int64(7))

	// Developer B - migration 6
	// This should raise an error. By default goose does not allow out-of-order (missing)
	// migrations, which means halt if a missing migration is detected.
	err = goose.Up(db, migrationsDir)
	is.True(err != nil) // error: found 1 missing migrations
	is.True(strings.Contains(err.Error(), "missing migrations"))
	// Confirm db version is unchanged.
	current, err = goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(current, int64(7))
}

func TestAllowMissingUp(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// Create and apply first 5 migrations.
	db := setupAllowMissingTestDB(t, 5)

	/*
		Developer A and B simultaneously check out the "main" currently on version 5.
		Developer A mistakenly creates migration 7 and commits.
		Developer B did not pull the latest changes and commits migration 6. Oops.

		If goose is set to allow missing migrations, then 6 should be applied
		after 7.
	*/

	// Developer A - migration 7 (mistakenly applied)
	{
		migrations, err := goose.CollectMigrations(migrationsDir, 0, 7)
		is.NoErr(err)
		err = migrations[6].Up(db)
		is.NoErr(err)
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(7))
	}
	// Developer B - migration 6
	{
		// By default, attempting to apply this migration will raise an error.
		// If goose is set to "allow missing" migrations then it should get applied.
		err := goose.Up(db, migrationsDir, goose.WithAllowMissing())
		is.True(err == nil) // Applying out-of-order migration should return no error when allow-missing=true
		count, err := getGooseVersionCount(db, goose.TableName())
		is.NoErr(err)
		is.Equal(count, int64(8)) // Expecting count of migrations to be 8

		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(8)) // Expecting max(version_id) to be 8
	}
}

func TestNowAllowMissingUpByOne(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// Create and apply first 5 migrations.
	db := setupAllowMissingTestDB(t, 5)

	/*
		Developer A and B simultaneously check out the "main" currently on version 5.
		Developer A mistakenly creates migration 7 and commits.
		Developer B did not pull the latest changes and commits migration 6. Oops.

		If goose is set to allow missing migrations, then 6 should be applied
		after 7.
	*/

	// Developer A - migration 7 (mistakenly applied)
	{
		migrations, err := goose.CollectMigrations(migrationsDir, 0, 7)
		is.NoErr(err)
		err = migrations[6].Up(db)
		is.NoErr(err)
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(7))
	}
	// Developer B - migration 6
	{
		// By default, this should raise an error.
		err := goose.UpByOne(db, migrationsDir)
		is.True(err != nil) // error: found 1 missing migrations
		is.True(strings.Contains(err.Error(), "missing migrations"))

		count, err := getGooseVersionCount(db, goose.TableName())
		is.NoErr(err)
		is.Equal(count, int64(6)) // Expecting count of migrations to be 6

		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(7)) // Expecting max(version_id) to be 7
	}
}

func TestAllowMissingUpByOne(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// Create and apply first 5 migrations.
	db := setupAllowMissingTestDB(t, 5)

	/*
		Developer A and B simultaneously check out the "main" currently on version 5.
		Developer A mistakenly creates migration 7 and commits.
		Developer B did not pull the latest changes and commits migration 6. Oops.

		If goose is set to allow missing migrations, then 6 should be applied
		after 7.
	*/

	// Developer A - migration 7 (mistakenly applied)
	{
		migrations, err := goose.CollectMigrations(migrationsDir, 0, 7)
		is.NoErr(err)
		err = migrations[6].Up(db)
		is.NoErr(err)
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(7))
	}
	// Developer B - migration 6
	{
		// By default, this should raise an error.
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		is.True(err == nil)

		count, err := getGooseVersionCount(db, goose.TableName())
		is.NoErr(err)
		is.Equal(count, int64(7)) // Expecting count of migrations to be 7

		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(6)) // Expecting max(version_id) to be 6
	}
	// Developer B - migration 8
	{
		// By default, this should raise an error.
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		is.True(err == nil)

		count, err := getGooseVersionCount(db, goose.TableName())
		is.NoErr(err)
		is.Equal(count, int64(8)) // Expecting count of migrations to be 8

		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(8)) // Expecting max(version_id) to be 8
	}
}

// setupAllowMissingTestDB is helper to setup a DB and apply migrations to
// a specific version.
func setupAllowMissingTestDB(t *testing.T, version int64) *sql.DB {
	is := is.New(t)
	db, err := newDockerDB(t)
	is.NoErr(err)

	// This is boilerplate to get the DB state to a specific migration.

	goose.SetDialect(*dialect)
	// Create goose table.
	current, err := goose.EnsureDBVersion(db)
	is.NoErr(err)
	is.True(current == int64(0))
	// Collect first 5 migrations.
	migrations, err := goose.CollectMigrations(migrationsDir, 0, version)
	is.NoErr(err)
	is.True(int64(len(migrations)) == version)
	// Apply first n migrations manually.
	for _, m := range migrations {
		err := m.Up(db)
		is.NoErr(err)
	}
	// Verify the current DB version is the Nth migration. This will only
	// work for sqeuentially applied migrations.
	current, err = goose.GetDBVersion(db)
	is.NoErr(err)
	is.True(current == int64(version))

	return db
}
