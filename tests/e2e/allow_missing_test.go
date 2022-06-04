package e2e

import (
	"database/sql"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestNotAllowMissing(t *testing.T) {

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)

	// Developer A and B check out the "main" branch which is currently
	// on version 5. Developer A mistakenly creates migration 7 and commits.
	// Developer B did not pull the latest changes and commits migration 6. Oops.

	// Developer A - migration 7 (mistakenly applied)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, 7)
	check.NoError(t, err)
	err = migrations[6].Up(db)
	check.NoError(t, err)
	current, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, current, 7)

	// Developer B - migration 6 (missing) and 8 (new)
	// This should raise an error. By default goose does not allow missing (out-of-order)
	// migrations, which means halt if a missing migration is detected.
	err = goose.Up(db, migrationsDir)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "missing migrations")
	// Confirm db version is unchanged.
	current, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, current, 7)
}

func TestAllowMissingUpWithRedo(t *testing.T) {

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)

	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	if len(migrations) == 0 {
		t.Fatalf("got zero migrations")
	}

	// Migration 7
	{
		migrations, err := goose.CollectMigrations(migrationsDir, 0, 7)
		check.NoError(t, err)
		err = migrations[6].Up(db)
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, current, 7)

		// Redo the previous Up migration and re-apply it.
		err = goose.Redo(db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migrations[6].Version)
	}
	// Migration 6
	{
		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, 6)

		err = goose.Redo(db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err = goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, 6)
	}
}

func TestNowAllowMissingUpByOne(t *testing.T) {

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
		check.NoError(t, err)
		err = migrations[6].Up(db)
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, current, 7)
	}
	// Developer B - migration 6
	{
		// By default, this should raise an error.
		err := goose.UpByOne(db, migrationsDir)
		// error: found 1 missing migrations
		check.HasError(t, err)
		check.Contains(t, err.Error(), "missing migrations")

		count, err := getGooseVersionCount(db, goose.TableName())
		check.NoError(t, err)
		check.Number(t, count, 6)

		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		// Expecting max(version_id) to be 7
		check.Number(t, current, 7)
	}
}

func TestAllowMissingUpWithReset(t *testing.T) {

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
		check.NoError(t, err)
		err = migrations[6].Up(db)
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, current, 7)
	}
	// Developer B - migration 6 (missing) and 8 (new)
	{
		// By default, attempting to apply this migration will raise an error.
		// If goose is set to "allow missing" migrations then it should get applied.
		err := goose.Up(db, migrationsDir, goose.WithAllowMissing())
		// Applying missing migration should return no error when allow-missing=true
		check.NoError(t, err)

		// Avoid hard-coding total and max, instead resolve it from the testdata migrations.
		// In other words, we applied 1..5,7,6,8 and this test shouldn't care
		// about migration 9 and onwards.
		allMigrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
		check.NoError(t, err)
		maxVersionID := allMigrations[len(allMigrations)-1].Version

		count, err := getGooseVersionCount(db, goose.TableName())
		check.NoError(t, err)
		// Count should be all testdata migrations (all applied)
		check.Number(t, count, len(allMigrations))

		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		// Expecting max(version_id) to be highest version in testdata
		check.Number(t, current, maxVersionID)
	}

	// Migrate everything down using Reset.
	err := goose.Reset(db, migrationsDir)
	check.NoError(t, err)
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)
}

func TestAllowMissingUpByOne(t *testing.T) {

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
		check.NoError(t, err)
		err = migrations[6].Up(db)
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, current, 7)
	}
	// Developer B - migration 6
	{
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		check.NoError(t, err)

		count, err := getGooseVersionCount(db, goose.TableName())
		check.NoError(t, err)
		// Expecting count of migrations to be 7
		check.Number(t, count, 7)

		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		// Expecting max(version_id) to be 6
		check.Number(t, current, 6)
	}
	// Developer B - migration 8
	{
		// By default, this should raise an error.
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		check.NoError(t, err)

		count, err := getGooseVersionCount(db, goose.TableName())
		check.NoError(t, err)
		// Expecting count of migrations to be 8
		check.Number(t, count, 8)

		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		// Expecting max(version_id) to be 8
		check.Number(t, current, 8)
	}
}

func TestMigrateAllowMissingDown(t *testing.T) {

	const (
		maxVersion = 8
	)
	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)

	// Developer A - migration 7 (mistakenly applied)
	{
		migrations, err := goose.CollectMigrations(migrationsDir, 0, maxVersion-1)
		check.NoError(t, err)
		err = migrations[6].Up(db)
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, current, maxVersion-1)
	}
	// Developer B - migration 6 (missing) and 8 (new)
	{
		// 6
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		check.NoError(t, err)
		// 8
		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		check.NoError(t, err)

		count, err := getGooseVersionCount(db, goose.TableName())
		check.NoError(t, err)
		check.Number(t, count, maxVersion)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		// Expecting max(version_id) to be 8
		check.Number(t, current, maxVersion)
	}
	// The order in the database is expected to be:
	// 1,2,3,4,5,7,6,8
	// So migrating down should be the reverse order:
	// 8,6,7,5,4,3,2,1
	//
	// Migrate down by one. Expecting 6.
	{
		err := goose.Down(db, migrationsDir)
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		// Expecting max(version) to be 6
		check.Number(t, current, 6)
	}
	// Migrate down by one. Expecting 7.
	{
		err := goose.Down(db, migrationsDir)
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		// Expecting max(version) to be 7
		check.Number(t, current, 7)
	}
	// Migrate down by one. Expecting 5.
	{
		err := goose.Down(db, migrationsDir)
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		// Expecting max(version) to be 5
		check.Number(t, current, 5)
	}
}

// setupTestDB is helper to setup a DB and apply migrations
// up to the specified version.
func setupTestDB(t *testing.T, version int64) *sql.DB {
	db, err := newDockerDB(t)
	check.NoError(t, err)

	goose.SetDialect(*dialect)

	// Create goose table.
	current, err := goose.EnsureDBVersion(db)
	check.NoError(t, err)
	check.Number(t, current, 0)
	// Collect first 5 migrations.
	migrations, err := goose.CollectMigrations(migrationsDir, 0, version)
	check.NoError(t, err)
	check.Number(t, len(migrations), version)
	// Apply n migrations manually.
	for _, m := range migrations {
		err := m.Up(db)
		check.NoError(t, err)
	}
	// Verify the current DB version is the Nth migration. This will only
	// work for sequentially applied migrations.
	current, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, current, version)

	return db
}
