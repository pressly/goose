package postgres_test

import (
	"context"
	"math"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
)

// Developer A and B check out the "main" branch which is currently on version 5. Developer A
// mistakenly creates migration 7 and commits. Developer B did not pull the latest changes and
// commits migration 6. Oops -- now the migrations are out of order.
//
// When goose is set to allow missing migrations, then 6 is applied after 7 with no error.

func TestNotAllowMissing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create and apply first 5 migrations.
	te := newTestEnv(t, migrationsDir, nil)
	_, err := te.provider.UpTo(ctx, 5)
	check.NoError(t, err)
	currentVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, 5)

	// Developer A - migration 7 (mistakenly applied)
	result, err := te.provider.ApplyVersion(context.Background(), 7, true)
	check.NoError(t, err)
	check.Number(t, result.Version, 7)
	current, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, current, 7)

	// Developer B - migration 6 (missing) and 8 (new). This should raise an error. By default goose
	// does not allow missing (out-of-order) migrations, which means halt if a missing migration is
	// detected.
	_, err = te.provider.Up(ctx)
	check.HasError(t, err)
	// found 1 missing (out-of-order) migration: [00006_f.sql]
	check.Contains(t, err.Error(), "missing (out-of-order) migration")
	// Confirm db version is unchanged.
	current, err = te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, current, 7)

	_, err = te.provider.UpByOne(ctx)
	check.HasError(t, err)
	// found 1 missing (out-of-order) migration: [00006_f.sql]
	check.Contains(t, err.Error(), "missing (out-of-order) migration")
	// Confirm db version is unchanged.
	current, err = te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, current, 7)

	_, err = te.provider.UpTo(ctx, math.MaxInt64)
	check.HasError(t, err)
	// found 1 missing (out-of-order) migration: [00006_f.sql]
	check.Contains(t, err.Error(), "missing (out-of-order) migration")
	// Confirm db version is unchanged.
	current, err = te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, current, 7)
}

func TestAllowMissingUpWithRedo(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create and apply first 5 migrations.
	te := newTestEnv(t, migrationsDir, nil)
	_, err := te.provider.UpTo(ctx, 5)
	check.NoError(t, err)
	currentVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, 5)

	// Initialize a new goose provider with the same db connection, but with allow missing set to
	// true.
	defaultOptions := goose.DefaultOptions().
		SetVerbose(testing.Verbose()).
		SetDir(migrationsDir).
		SetAllowMissing(true)

	// Developer A - migration 7 (mistakenly applied)
	{
		p, err := goose.NewProvider(
			goose.DialectPostgres,
			te.db,
			// exclude migration 6 because it doesn't exist yet on the filesystem.
			defaultOptions.SetExcludeFilenames("00006_f.sql"),
		)
		check.NoError(t, err)

		_, err = p.ApplyVersion(ctx, 7, true)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 7)

		// Redo the previous Up migration and re-apply it.
		redoResult, err := p.Redo(ctx)
		check.NoError(t, err)
		check.Number(t, len(redoResult), 2)
		check.Number(t, redoResult[0].Version, 7)
		check.Number(t, redoResult[1].Version, 7)
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 7)
	}
	// Developer B - migration 6 (missing).
	{
		p, err := goose.NewProvider(goose.DialectPostgres, te.db, defaultOptions)
		check.NoError(t, err)

		_, err = p.UpByOne(ctx)
		check.NoError(t, err)
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 6)

		redoResult, err := p.Redo(ctx)
		check.NoError(t, err)
		check.Number(t, len(redoResult), 2)
		check.Number(t, redoResult[0].Version, 6)
		check.Number(t, redoResult[1].Version, 6)
		currentVersion, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 6)

		// Developer C - migration 8 (new).
		_, err = p.UpByOne(ctx)
		check.NoError(t, err)
		count, err := getGooseVersionCount(te.db, defaultOptions.TableName)
		check.NoError(t, err)
		// Expecting count of migrations to be 8
		check.Number(t, count, 8)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version_id) to be 8
		check.Number(t, current, 8)
	}
}

func TestAllowMissingUpWithReset(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Developer A and B check out the "main" branch which is currently on version 5. Developer A
	// mistakenly creates migration 7 and commits. Developer B did not pull the latest changes and
	// commits migration 6. Oops.
	//
	// When goose is set to allow missing migrations, then 6 is applied after 7 with no error.

	// Create and apply first 5 migrations.
	te := newTestEnv(t, migrationsDir, nil)
	_, err := te.provider.UpTo(ctx, 5)
	check.NoError(t, err)
	currentVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, 5)

	// Initialize a new goose provider with the same db connection, but with allow missing set to
	// true.
	defaultOptions := goose.DefaultOptions().
		SetVerbose(testing.Verbose()).
		SetDir(migrationsDir).
		SetAllowMissing(true)

	// Developer A - migration 7 (mistakenly applied)
	{
		p, err := goose.NewProvider(
			goose.DialectPostgres,
			te.db,
			// exclude migration 6 because it doesn't exist yet on the filesystem.
			defaultOptions.SetExcludeFilenames("00006_f.sql"),
		)
		check.NoError(t, err)

		_, err = p.ApplyVersion(ctx, 7, true)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 7)
	}
	// Developer B - migration 6 (missing) and 8,9,10,11 (new)
	{
		p, err := goose.NewProvider(goose.DialectPostgres, te.db, defaultOptions)
		check.NoError(t, err)

		upResult, err := p.Up(ctx)
		check.NoError(t, err)
		check.Number(t, len(upResult), 5)
		expected := []int64{6, 8, 9, 10, 11}
		for i := range upResult {
			check.Number(t, upResult[i].Version, expected[i])
		}
		all := p.ListSources()

		count, err := getGooseVersionCount(te.db, defaultOptions.TableName)
		check.NoError(t, err)
		// Count should be all testdata migrations (all applied)
		check.Number(t, count, len(all))

		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		// Expecting max(version_id) to be highest version in testdata
		check.Number(t, current, lastVersion(p.ListSources()))
	}
	// Migrate everything down using Reset.
	{
		p, err := goose.NewProvider(goose.DialectPostgres, te.db, defaultOptions)
		check.NoError(t, err)
		resetResults, err := p.DownTo(ctx, 0)
		check.NoError(t, err)
		check.Number(t, len(resetResults), 11)
		expected := []int64{11, 10, 9, 8, 6, 7, 5, 4, 3, 2, 1}
		for i := range resetResults {
			check.Number(t, resetResults[i].Version, expected[i])
		}
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 0)
	}
}

func TestMigrateAllowMissingDown(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Create and apply first 5 migrations.
	te := newTestEnv(t, migrationsDir, nil)
	_, err := te.provider.UpTo(ctx, 5)
	check.NoError(t, err)
	currentVersion, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, currentVersion, 5)

	defaultOptions := goose.DefaultOptions().
		SetVerbose(testing.Verbose()).
		SetDir(migrationsDir).
		SetAllowMissing(true)

	// Developer A - migration 7 (mistakenly applied)
	{
		p, err := goose.NewProvider(
			goose.DialectPostgres,
			te.db,
			// exclude migration 6 because it doesn't exist yet on the filesystem.
			defaultOptions.SetExcludeFilenames("00006_f.sql"),
		)
		check.NoError(t, err)

		_, err = p.ApplyVersion(ctx, 7, true)
		check.NoError(t, err)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 7)
	}
	// Developer B - migration 6 (missing) and 8 (new)
	{
		p, err := goose.NewProvider(goose.DialectPostgres, te.db, defaultOptions)
		check.NoError(t, err)
		// 6
		upResult, err := p.UpByOne(ctx)
		check.NoError(t, err)
		check.Number(t, upResult.Version, 6)
		// 8
		upResult, err = p.UpByOne(ctx)
		check.NoError(t, err)
		check.Number(t, upResult.Version, 8)

		count, err := getGooseVersionCount(te.db, defaultOptions.TableName)
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

	p, err := goose.NewProvider(goose.DialectPostgres, te.db, defaultOptions)
	check.NoError(t, err)
	expected := []int64{8, 6, 7, 5, 4, 3, 2, 1, 0}
	for i, v := range expected {
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, v)
		downResult, err := p.Down(ctx)
		if i == len(expected)-1 {
			check.HasError(t, goose.ErrVersionNotFound)
		} else {
			check.NoError(t, err)
			check.Number(t, downResult.Version, v)
		}
	}
}

func TestWithWithoutAllowMissing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	// Test for https://github.com/pressly/goose/issues/521

	// Apply 1,2,4,3 then run up without allow missing. If the next requested migration is
	// 4 then it should not raise an error because it has already been applied.
	// If goose attempts to apply 4 again then it will raise an error (SQLSTATE 42701) because it
	// has already been applied. Specifically it will raise a  error.
	// Create and apply first 5 migrations.
	opt := goose.DefaultOptions().
		SetAllowMissing(true)
	te := newTestEnv(t, migrationsDir, &opt)
	_, err := te.provider.UpTo(ctx, 2)
	check.NoError(t, err)
	_, err = te.provider.ApplyVersion(ctx, 4, true)
	check.NoError(t, err)
	res, err := te.provider.UpTo(ctx, 4)
	check.NoError(t, err)
	check.Number(t, len(res), 1)
	check.Number(t, res[0].Version, 3)

	{
		// Create a new goose provider WITHOUT allow missing, and attempt to apply migration 4.
		opt := goose.DefaultOptions().
			SetDir(migrationsDir).
			SetVerbose(testing.Verbose())
		provider, err := goose.NewProvider(goose.DialectPostgres, te.db, opt)
		check.NoError(t, err)
		results, err := provider.UpTo(ctx, 4)
		check.NoError(t, err)
		check.Number(t, len(results), 0)
		// And now try to apply the next migration in the sequence.
		result, err := provider.UpByOne(ctx)
		check.NoError(t, err)
		check.Number(t, result.Version, 5)
	}
}
