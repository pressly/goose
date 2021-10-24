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
	db := setupTestDB(t, 5)

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

	// Developer B - migration 6 (missing) and 8 (new)
	// This should raise an error. By default goose does not allow missing (out-of-order)
	// migrations, which means halt if a missing migration is detected.
	err = goose.Up(db, migrationsDir)
	is.True(err != nil) // error: found 1 missing migrations
	is.True(strings.Contains(err.Error(), "missing migrations"))
	// Confirm db version is unchanged.
	current, err = goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(current, int64(7))
}

func TestAllowMissingUpWithRedo(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)

	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	is.NoErr(err)
	is.True(len(migrations) != 0)

	// Migration 7
	{
		migrations, err := goose.CollectMigrations(migrationsDir, 0, 7)
		is.NoErr(err)
		err = migrations[6].Up(db)
		is.NoErr(err)
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(7))

		// Redo the previous Up migration and re-apply it.
		err = goose.Redo(db, migrationsDir)
		is.NoErr(err)
		currentVersion, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.True(currentVersion == migrations[6].Version)
	}
	// Migration 6
	{
		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		is.NoErr(err)
		currentVersion, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(currentVersion, int64(6))

		err = goose.Redo(db, migrationsDir)
		is.NoErr(err)
		currentVersion, err = goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(currentVersion, int64(6))
	}
}

func TestNowAllowMissingUpByOne(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)

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

func TestAllowMissingUpWithReset(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)

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
	// Developer B - migration 6 (missing) and 8 (new)
	{
		// By default, attempting to apply this migration will raise an error.
		// If goose is set to "allow missing" migrations then it should get applied.
		err := goose.Up(db, migrationsDir, goose.WithAllowMissing())
		is.NoErr(err) // Applying missing migration should return no error when allow-missing=true

		// Avoid hard-coding total and max, instead resolve it from the testdata migrations.
		// In other words, we applied 1..5,7,6,8 and this test shouldn't care
		// about migration 9 and onwards.
		allMigrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
		is.NoErr(err)
		maxVersionID := allMigrations[len(allMigrations)-1].Version

		count, err := getGooseVersionCount(db, goose.TableName())
		is.NoErr(err)
		is.Equal(count, int64(len(allMigrations))) // Count should be all testdata migrations (all applied)

		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, maxVersionID) // Expecting max(version_id) to be highest version in testdata
	}

	// Migrate everything down using Reset.
	err := goose.Reset(db, migrationsDir)
	is.NoErr(err)
	currentVersion, err := goose.GetDBVersion(db)
	is.NoErr(err)
	is.Equal(currentVersion, int64(0))
}

func TestAllowMissingUpByOne(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)

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
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		is.NoErr(err)

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
		is.NoErr(err)

		count, err := getGooseVersionCount(db, goose.TableName())
		is.NoErr(err)
		is.Equal(count, int64(8)) // Expecting count of migrations to be 8

		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(8)) // Expecting max(version_id) to be 8
	}
}

func TestMigrateAllowMissingDown(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	const (
		maxVersion = 8
	)
	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)

	// Developer A - migration 7 (mistakenly applied)
	{
		migrations, err := goose.CollectMigrations(migrationsDir, 0, maxVersion-1)
		is.NoErr(err)
		err = migrations[6].Up(db)
		is.NoErr(err)
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(maxVersion-1))
	}
	// Developer B - migration 6 (missing) and 8 (new)
	{
		// 6
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		is.NoErr(err)
		// 8
		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		is.NoErr(err)

		count, err := getGooseVersionCount(db, goose.TableName())
		is.NoErr(err)
		is.Equal(count, int64(maxVersion)) // Expecting count of migrations to be 8
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(maxVersion)) // Expecting max(version_id) to be 8
	}
	// The order in the database is expected to be:
	// 1,2,3,4,5,7,6,8
	// So migrating down should be the reverse order:
	// 8,6,7,5,4,3,2,1
	//
	// Migrate down by one. Expecting 6.
	{
		err := goose.Down(db, migrationsDir)
		is.NoErr(err)
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(6)) // Expecting max(version) to be 6
	}
	// Migrate down by one. Expecting 7.
	{
		err := goose.Down(db, migrationsDir)
		is.NoErr(err)
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(7)) // Expecting max(version) to be 7
	}
	// Migrate down by one. Expecting 5.
	{
		err := goose.Down(db, migrationsDir)
		is.NoErr(err)
		current, err := goose.GetDBVersion(db)
		is.NoErr(err)
		is.Equal(current, int64(5)) // Expecting max(version) to be 5
	}
}

// setupTestDB is helper to setup a DB and apply migrations
// up to the specified version.
func setupTestDB(t *testing.T, version int64) *sql.DB {
	is := is.New(t)
	db, err := newDockerDB(t)
	is.NoErr(err)

	goose.SetDialect(*dialect)

	// Create goose table.
	current, err := goose.EnsureDBVersion(db)
	is.NoErr(err)
	is.True(current == int64(0))
	// Collect first 5 migrations.
	migrations, err := goose.CollectMigrations(migrationsDir, 0, version)
	is.NoErr(err)
	is.True(int64(len(migrations)) == version)
	// Apply n migrations manually.
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
