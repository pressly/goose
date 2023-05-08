package e2e

import (
	"errors"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestMigrateUpWithResetDryRun(t *testing.T) {
	t.Parallel()

	db, err := newDockerDB(t)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))

	// Migrate all with a dry run.
	err = goose.Up(db, migrationsDir, goose.WithDryRun())
	check.NoError(t, err)
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)

	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	// incorrect database version
	check.Number(t, gotVersion, currentVersion)

	// Migrate all, for real this time.
	err = goose.Up(db, migrationsDir)
	check.NoError(t, err)
	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, migrations[len(migrations)-1].Version)

	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err = getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	// incorrect database version
	check.Number(t, gotVersion, currentVersion)

	// Migrate everything down using Reset with dry run.
	err = goose.Reset(db, migrationsDir, goose.WithDryRun())
	check.NoError(t, err)
	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, migrations[len(migrations)-1].Version)

	// Migrate everything down using Reset, for real this time.
	err = goose.Reset(db, migrationsDir)
	check.NoError(t, err)
	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)
}

func TestMigrateUpWithRedoDryRun(t *testing.T) {
	t.Parallel()

	db, err := newDockerDB(t)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)

	check.NumberNotZero(t, len(migrations))
	startingVersion, err := goose.EnsureDBVersion(db)
	check.NoError(t, err)
	check.Number(t, startingVersion, 0)
	// Migrate all
	for _, migration := range migrations {
		err = migration.Up(db)
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migration.Version)

		// Redo the previous Up migration and re-apply it, with dry run.
		err = goose.Redo(db, migrationsDir, goose.WithDryRun())
		check.NoError(t, err)
		currentVersion, err = goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migration.Version)

		// Redo the previous Up migration and re-apply it, for real this time.
		err = goose.Redo(db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err = goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, migration.Version)
	}
	// Once everything is tested the version should match the highest testdata version
	maxVersion := migrations[len(migrations)-1].Version
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, maxVersion)
}

func TestMigrateUpToWithDryRun(t *testing.T) {
	t.Parallel()

	const (
		upToVersion int64 = 2
	)
	db, err := newDockerDB(t)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))

	// Migrate up to the second migration with dry run.
	err = goose.UpTo(db, migrationsDir, upToVersion, goose.WithDryRun())
	check.NoError(t, err)
	// Fetch the goose version from DB
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)
	// Validate the version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, 0) // incorrect database version

	// Migrate up to the second migration, for real this time.
	err = goose.UpTo(db, migrationsDir, upToVersion)
	check.NoError(t, err)
	// Fetch the goose version from DB
	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, upToVersion)
	// Validate the version actually matches what goose claims it is
	gotVersion, err = getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, upToVersion) // incorrect database version
}

func TestMigrateUpByOneWithDryRun(t *testing.T) {
	t.Parallel()

	db, err := newDockerDB(t)
	check.NoError(t, err)
	migrations, err := goose.CollectMigrations(migrationsDir, 0, goose.MaxVersion)
	check.NoError(t, err)
	check.NumberNotZero(t, len(migrations))
	// Apply all migrations one-by-one, first with dry run enabled, and then
	// for real.
	var counter int
	for {
		err := goose.UpByOne(db, migrationsDir, goose.WithDryRun())
		counter++
		if counter > len(migrations) {
			if !errors.Is(err, goose.ErrNoNextVersion) {
				t.Fatalf("incorrect error: got:%v want:%v", err, goose.ErrNoNextVersion)
			}
			break
		}
		check.NoError(t, err)

		err = goose.UpByOne(db, migrationsDir)
		check.NoError(t, err)
	}
	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, migrations[len(migrations)-1].Version)
	check.Number(t, migrations[counter-2].Version, currentVersion)
	// Validate the db migration version actually matches what goose claims it is
	gotVersion, err := getCurrentGooseVersion(db, goose.TableName())
	check.NoError(t, err)
	check.Number(t, gotVersion, currentVersion) // incorrect database version
}

func TestNotAllowMissingWithDryRun(t *testing.T) {
	t.Parallel()

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
	err = goose.Up(db, migrationsDir, goose.WithDryRun())
	check.HasError(t, err)
	check.Contains(t, err.Error(), "missing migrations")
	// Confirm db version is unchanged.
	current, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, current, 7)
}

func TestAllowMissingUpWithRedoWithDryRun(t *testing.T) {
	t.Parallel()

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

		// First, apply the migration with dry run.
		err = migrations[6].Up(db, goose.MigrationWithDryRun())
		check.NoError(t, err)
		current, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, current, 5)

		// Then, apply for real.
		err = migrations[6].Up(db)
		check.NoError(t, err)
		current, err = goose.GetDBVersion(db)
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
		// First, apply with allow missing with dry run.
		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing(), goose.WithDryRun())
		check.NoError(t, err)
		currentVersion, err := goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, 7)

		// Then, apply for real.
		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		check.NoError(t, err)
		currentVersion, err = goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, 6)

		err = goose.Redo(db, migrationsDir)
		check.NoError(t, err)
		currentVersion, err = goose.GetDBVersion(db)
		check.NoError(t, err)
		check.Number(t, currentVersion, 6)
	}
}

func TestNowAllowMissingUpByOneWithDryRun(t *testing.T) {
	t.Parallel()

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
		// By default, this should raise an error, even with dry run.
		err := goose.UpByOne(db, migrationsDir, goose.WithDryRun())
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

func TestAllowMissingUpWithResetWithDryRun(t *testing.T) {
	t.Parallel()

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
		err := goose.Up(db, migrationsDir, goose.WithAllowMissing(), goose.WithDryRun())
		// Applying missing migration should return no error when allow-missing=true
		check.NoError(t, err)

		// Perform again, for real this time.
		err = goose.Up(db, migrationsDir, goose.WithAllowMissing())
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

	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)

	// Migrate everything down using Reset, first with dry run and then for
	// real.
	err = goose.Reset(db, migrationsDir, goose.WithDryRun())
	check.NoError(t, err)
	newCurrentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, newCurrentVersion, currentVersion)

	err = goose.Reset(db, migrationsDir)
	check.NoError(t, err)
	newCurrentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, newCurrentVersion, 0)
}

func TestAllowMissingUpByOneWithDryRun(t *testing.T) {
	t.Parallel()

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
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing(), goose.WithDryRun())
		check.NoError(t, err)

		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
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

func TestMigrateAllowMissingDownWithDryRun(t *testing.T) {
	t.Parallel()

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
		err := goose.UpByOne(db, migrationsDir, goose.WithAllowMissing(), goose.WithDryRun())
		check.NoError(t, err)
		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing())
		check.NoError(t, err)
		// 8
		err = goose.UpByOne(db, migrationsDir, goose.WithAllowMissing(), goose.WithDryRun())
		check.NoError(t, err)
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
