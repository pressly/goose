package e2e

import (
	"context"
	"database/sql"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
)

func TestNotAllowMissing(t *testing.T) {
	t.Parallel()

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)
	ctx := context.Background()

	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)
	// Developer A and B check out the "main" branch which is currently
	// on version 5. Developer A mistakenly creates migration 7 and commits.
	// Developer B did not pull the latest changes and commits migration 6. Oops.

	// Developer A - migration 7 (mistakenly applied)
	err = p.Apply(ctx, 7)
	check.NoError(t, err)
	dbVersion, err := p.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, dbVersion, 7)

	// Developer B - migration 6 (missing) and 8 (new)
	// This should raise an error. By default goose does not allow missing (out-of-order)
	// migrations, which means halt if a missing migration is detected.
	err = p.Up(ctx)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "missing migrations")
	// Confirm db version is unchanged.
	dbVersion, err = p.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, dbVersion, 7)
}

func TestAllowMissingUpWithRedo(t *testing.T) {
	t.Parallel()

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)
	ctx := context.Background()

	options := &goose.Options{
		AllowMissing: true,
	}
	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, options)
	check.NoError(t, err)

	// Migration 7
	{
		err := p.Apply(ctx, 7)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 7)

		// Redo the previous Up migration and re-apply it.
		err = p.Redo(ctx)
		check.NoError(t, err)
		dbVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, dbVersion, 7)
	}
	// Migration 6
	{
		err := p.UpByOne(ctx)
		check.NoError(t, err)
		dbVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, dbVersion, 6)

		err = p.Redo(ctx)
		check.NoError(t, err)
		dbVersion, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, dbVersion, 6)
	}
}

func TestNotAllowMissingUpByOne(t *testing.T) {
	t.Parallel()

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)
	ctx := context.Background()

	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)

	/*
		Developer A and B simultaneously check out the "main" currently on version 5.
		Developer A mistakenly creates migration 7 and commits.
		Developer B did not pull the latest changes and commits migration 6. Oops.

		If goose is set to allow missing migrations, then 6 should be applied
		after 7.
	*/

	// Developer A - migration 7 (mistakenly applied)
	{
		err = p.Apply(ctx, 7)
		check.NoError(t, err)
		dbVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, dbVersion, 7)
	}
	// Developer B - migration 6
	{
		// By default, this should raise an error.
		err := p.UpByOne(ctx)
		// error: found 1 missing migrations
		check.HasError(t, err)
		check.Contains(t, err.Error(), "missing migrations")

		count, err := getGooseVersionCount(db, defaultTableName)
		check.NoError(t, err)
		check.Number(t, count, 6)

		dbVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version_id) to be 7
		check.Number(t, dbVersion, 7)
	}
}

func TestAllowMissingUpWithReset(t *testing.T) {
	t.Parallel()

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)
	ctx := context.Background()

	options := &goose.Options{
		AllowMissing: true,
	}
	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, options)
	check.NoError(t, err)

	/*
		Developer A and B simultaneously check out the "main" currently on version 5.
		Developer A mistakenly creates migration 7 and commits.
		Developer B did not pull the latest changes and commits migration 6. Oops.

		If goose is set to allow missing migrations, then 6 should be applied
		after 7.
	*/

	// Developer A - migration 7 (mistakenly applied)
	{
		err := p.Apply(ctx, 7)
		check.NoError(t, err)
		dbVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, dbVersion, 7)
	}
	// Developer B - migration 6 (missing) and 8 (new)
	{
		// The goose provider is set to apply migrations out-of-order, so this
		// should not raise an error.
		err := p.Up(ctx)
		check.NoError(t, err)

		// Avoid hard-coding total and max, instead resolve it from the testdata migrations.
		// In other words, we applied 1..5,7,6,8 and this test shouldn't care
		// about migration 9 and onwards.
		allMigrations := p.ListMigrations()
		maxVersionID := allMigrations[len(allMigrations)-1].Version

		count, err := getGooseVersionCount(db, defaultTableName)
		check.NoError(t, err)
		// Count should be all testdata migrations (all applied)
		check.Number(t, count, len(allMigrations))

		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version_id) to be highest version in testdata
		check.Number(t, current, maxVersionID)
	}

	// Migrate everything down using Reset.
	err = p.Reset(ctx)
	check.NoError(t, err)
	dbVersion, err := p.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, dbVersion, 0)
}

func TestAllowMissingUpByOne(t *testing.T) {
	t.Parallel()

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)
	ctx := context.Background()

	options := &goose.Options{
		AllowMissing: true,
	}
	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, options)
	check.NoError(t, err)

	/*
		Developer A and B simultaneously check out the "main" currently on version 5.
		Developer A mistakenly creates migration 7 and commits.
		Developer B did not pull the latest changes and commits migration 6. Oops.

		If goose is set to allow missing migrations, then 6 should be applied
		after 7.
	*/

	// Developer A - migration 7 (mistakenly applied)
	{
		err = p.Apply(ctx, 7)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 7)
	}
	// Developer B - migration 6
	{
		err := p.UpByOne(ctx)
		check.NoError(t, err)

		count, err := getGooseVersionCount(db, defaultTableName)
		check.NoError(t, err)
		// Expecting count of migrations to be 7
		check.Number(t, count, 7)

		dbVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version_id) to be 6
		check.Number(t, dbVersion, 6)
	}
	// Developer B - migration 8
	{
		// By default, this should raise an error.
		err := p.UpByOne(ctx)
		check.NoError(t, err)

		count, err := getGooseVersionCount(db, defaultTableName)
		check.NoError(t, err)
		// Expecting count of migrations to be 8
		check.Number(t, count, 8)

		dbVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version_id) to be 8
		check.Number(t, dbVersion, 8)
	}
}

func TestMigrateAllowMissingDown(t *testing.T) {
	t.Parallel()

	// Create and apply first 5 migrations.
	db := setupTestDB(t, 5)
	ctx := context.Background()

	options := &goose.Options{
		AllowMissing: true,
	}
	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, options)
	check.NoError(t, err)

	// Developer A - migration 7 (mistakenly applied)
	{
		err := p.Apply(ctx, 7)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 7)
	}
	// Developer B - migration 6 (missing) and 8 (new)
	{
		// 6
		err := p.UpByOne(ctx)
		check.NoError(t, err)
		// 8
		err = p.UpByOne(ctx)
		check.NoError(t, err)

		count, err := getGooseVersionCount(db, defaultTableName)
		check.NoError(t, err)
		check.Number(t, count, 8)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version_id) to be 8
		check.Number(t, current, 8)
	}
	// The order in the database is expected to be:
	// 1,2,3,4,5,7,6,8
	// So migrating down should be the reverse order:
	// 8,6,7,5,4,3,2,1
	//
	// Migrate down by one. Expecting 6.
	{
		err := p.Down(ctx)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version) to be 6
		check.Number(t, current, 6)
	}
	// Migrate down by one. Expecting 7.
	{
		err := p.Down(ctx)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version) to be 7
		check.Number(t, current, 7)
	}
	// Migrate down by one. Expecting 5.
	{
		err := p.Down(ctx)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version) to be 5
		check.Number(t, current, 5)
	}
}

// setupTestDB is helper to setup a DB and apply migrations
// up to the specified version.
func setupTestDB(t *testing.T, version int64) *sql.DB {
	ctx := context.Background()
	db, err := newDockerDB(t)
	check.NoError(t, err)
	p, err := goose.NewProvider(toDialect(t, *dialect), db, migrationsDir, nil)
	check.NoError(t, err)
	err = p.UpTo(ctx, version)
	check.NoError(t, err)
	// Verify the currentVersion DB version is the Nth migration. This will only
	// work for sqeuentially applied migrations.
	dbVersion, err := p.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, dbVersion, version)
	return db
}
