package goose_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/check"
)

func TestProviderRun(t *testing.T) {
	t.Parallel()

	t.Run("closed_db", func(t *testing.T) {
		p, db := newProviderWithDB(t)
		check.NoError(t, db.Close())
		_, err := p.Up(context.Background())
		check.HasError(t, err)
		check.Equal(t, err.Error(), "failed to initialize: sql: database is closed")
	})
	t.Run("ping_and_close", func(t *testing.T) {
		p, _ := newProviderWithDB(t)
		t.Cleanup(func() {
			check.NoError(t, p.Close())
		})
		check.NoError(t, p.Ping(context.Background()))
	})
	t.Run("apply_unknown_version", func(t *testing.T) {
		p, _ := newProviderWithDB(t)
		_, err := p.ApplyVersion(context.Background(), 999, true)
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, goose.ErrVersionNotFound), true)
		_, err = p.ApplyVersion(context.Background(), 999, false)
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, goose.ErrVersionNotFound), true)
	})
	t.Run("run_zero", func(t *testing.T) {
		p, _ := newProviderWithDB(t)
		_, err := p.UpTo(context.Background(), 0)
		check.HasError(t, err)
		check.Equal(t, err.Error(), "version must be greater than 0")
		_, err = p.DownTo(context.Background(), -1)
		check.HasError(t, err)
		check.Equal(t, err.Error(), "invalid version: must be a valid number or zero: -1")
		_, err = p.ApplyVersion(context.Background(), 0, true)
		check.HasError(t, err)
		check.Equal(t, err.Error(), "version must be greater than 0")
	})
	t.Run("up_and_down_all", func(t *testing.T) {
		ctx := context.Background()
		p, _ := newProviderWithDB(t)
		const (
			numCount = 7
		)
		sources := p.ListSources()
		check.Number(t, len(sources), numCount)
		// Ensure only SQL migrations are returned
		for _, s := range sources {
			check.Equal(t, s.Type, goose.TypeSQL)
		}
		// Test Up
		res, err := p.Up(ctx)
		check.NoError(t, err)
		check.Number(t, len(res), numCount)
		assertResult(t, res[0], newSource(goose.TypeSQL, "00001_users_table.sql", 1), "up", false)
		assertResult(t, res[1], newSource(goose.TypeSQL, "00002_posts_table.sql", 2), "up", false)
		assertResult(t, res[2], newSource(goose.TypeSQL, "00003_comments_table.sql", 3), "up", false)
		assertResult(t, res[3], newSource(goose.TypeSQL, "00004_insert_data.sql", 4), "up", false)
		assertResult(t, res[4], newSource(goose.TypeSQL, "00005_posts_view.sql", 5), "up", false)
		assertResult(t, res[5], newSource(goose.TypeSQL, "00006_empty_up.sql", 6), "up", true)
		assertResult(t, res[6], newSource(goose.TypeSQL, "00007_empty_up_down.sql", 7), "up", true)
		// Test Down
		res, err = p.DownTo(ctx, 0)
		check.NoError(t, err)
		check.Number(t, len(res), numCount)
		assertResult(t, res[0], newSource(goose.TypeSQL, "00007_empty_up_down.sql", 7), "down", true)
		assertResult(t, res[1], newSource(goose.TypeSQL, "00006_empty_up.sql", 6), "down", true)
		assertResult(t, res[2], newSource(goose.TypeSQL, "00005_posts_view.sql", 5), "down", false)
		assertResult(t, res[3], newSource(goose.TypeSQL, "00004_insert_data.sql", 4), "down", false)
		assertResult(t, res[4], newSource(goose.TypeSQL, "00003_comments_table.sql", 3), "down", false)
		assertResult(t, res[5], newSource(goose.TypeSQL, "00002_posts_table.sql", 2), "down", false)
		assertResult(t, res[6], newSource(goose.TypeSQL, "00001_users_table.sql", 1), "down", false)
	})
	t.Run("up_and_down_by_one", func(t *testing.T) {
		ctx := context.Background()
		p, _ := newProviderWithDB(t)
		maxVersion := len(p.ListSources())
		// Apply all migrations one-by-one.
		var counter int
		for {
			res, err := p.UpByOne(ctx)
			counter++
			if counter > maxVersion {
				if !errors.Is(err, goose.ErrNoNextVersion) {
					t.Fatalf("incorrect error: got:%v want:%v", err, goose.ErrNoNextVersion)
				}
				break
			}
			check.NoError(t, err)
			check.Bool(t, res != nil, true)
			check.Number(t, res.Source.Version, int64(counter))
		}
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, int64(maxVersion))
		// Reset counter
		counter = 0
		// Rollback all migrations one-by-one.
		for {
			res, err := p.Down(ctx)
			counter++
			if counter > maxVersion {
				if !errors.Is(err, goose.ErrNoNextVersion) {
					t.Fatalf("incorrect error: got:%v want:%v", err, goose.ErrNoNextVersion)
				}
				break
			}
			check.NoError(t, err)
			check.Bool(t, res != nil, true)
			check.Number(t, res.Source.Version, int64(maxVersion-counter+1))
		}
		// Once everything is tested the version should match the highest testdata version
		currentVersion, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 0)
	})
	t.Run("up_to", func(t *testing.T) {
		ctx := context.Background()
		p, db := newProviderWithDB(t)
		const (
			upToVersion int64 = 2
		)
		results, err := p.UpTo(ctx, upToVersion)
		check.NoError(t, err)
		check.Number(t, len(results), upToVersion)
		assertResult(t, results[0], newSource(goose.TypeSQL, "00001_users_table.sql", 1), "up", false)
		assertResult(t, results[1], newSource(goose.TypeSQL, "00002_posts_table.sql", 2), "up", false)
		// Fetch the goose version from DB
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, upToVersion)
		// Validate the version actually matches what goose claims it is
		gotVersion, err := getMaxVersionID(db, goose.DefaultTablename)
		check.NoError(t, err)
		check.Number(t, gotVersion, upToVersion)
	})
	t.Run("sql_connections", func(t *testing.T) {
		tt := []struct {
			name         string
			maxOpenConns int
			maxIdleConns int
			useDefaults  bool
		}{
			// Single connection ensures goose is able to function correctly when multiple
			// connections are not available.
			{name: "single_conn", maxOpenConns: 1, maxIdleConns: 1},
			{name: "defaults", useDefaults: true},
		}
		for _, tc := range tt {
			t.Run(tc.name, func(t *testing.T) {
				ctx := context.Background()
				// Start a new database for each test case.
				p, db := newProviderWithDB(t)
				if !tc.useDefaults {
					db.SetMaxOpenConns(tc.maxOpenConns)
					db.SetMaxIdleConns(tc.maxIdleConns)
				}
				sources := p.ListSources()
				check.NumberNotZero(t, len(sources))

				currentVersion, err := p.GetDBVersion(ctx)
				check.NoError(t, err)
				check.Number(t, currentVersion, 0)

				{
					// Apply all up migrations
					upResult, err := p.Up(ctx)
					check.NoError(t, err)
					check.Number(t, len(upResult), len(sources))
					currentVersion, err := p.GetDBVersion(ctx)
					check.NoError(t, err)
					check.Number(t, currentVersion, p.ListSources()[len(sources)-1].Version)
					// Validate the db migration version actually matches what goose claims it is
					gotVersion, err := getMaxVersionID(db, goose.DefaultTablename)
					check.NoError(t, err)
					check.Number(t, gotVersion, currentVersion)
					tables, err := getTableNames(db)
					check.NoError(t, err)
					if !reflect.DeepEqual(tables, knownTables) {
						t.Logf("got tables: %v", tables)
						t.Logf("known tables: %v", knownTables)
						t.Fatal("failed to match tables")
					}
				}
				{
					// Apply all down migrations
					downResult, err := p.DownTo(ctx, 0)
					check.NoError(t, err)
					check.Number(t, len(downResult), len(sources))
					gotVersion, err := getMaxVersionID(db, goose.DefaultTablename)
					check.NoError(t, err)
					check.Number(t, gotVersion, 0)
					// Should only be left with a single table, the default goose table
					tables, err := getTableNames(db)
					check.NoError(t, err)
					knownTables := []string{goose.DefaultTablename, "sqlite_sequence"}
					if !reflect.DeepEqual(tables, knownTables) {
						t.Logf("got tables: %v", tables)
						t.Logf("known tables: %v", knownTables)
						t.Fatal("failed to match tables")
					}
				}
			})
		}
	})
	t.Run("apply", func(t *testing.T) {
		ctx := context.Background()
		p, _ := newProviderWithDB(t)
		sources := p.ListSources()
		// Apply all migrations in the up direction.
		for _, s := range sources {
			res, err := p.ApplyVersion(ctx, s.Version, true)
			check.NoError(t, err)
			// Round-trip the migration result through the database to ensure it's valid.
			var empty bool
			if s.Version == 6 || s.Version == 7 {
				empty = true
			}
			assertResult(t, res, s, "up", empty)
		}
		// Apply all migrations in the down direction.
		for i := len(sources) - 1; i >= 0; i-- {
			s := sources[i]
			res, err := p.ApplyVersion(ctx, s.Version, false)
			check.NoError(t, err)
			// Round-trip the migration result through the database to ensure it's valid.
			var empty bool
			if s.Version == 6 || s.Version == 7 {
				empty = true
			}
			assertResult(t, res, s, "down", empty)
		}
		// Try apply version 1 multiple times
		_, err := p.ApplyVersion(ctx, 1, true)
		check.NoError(t, err)
		_, err = p.ApplyVersion(ctx, 1, true)
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, goose.ErrAlreadyApplied), true)
		check.Contains(t, err.Error(), "version 1: migration already applied")
	})
	t.Run("status", func(t *testing.T) {
		ctx := context.Background()
		p, _ := newProviderWithDB(t)
		numCount := len(p.ListSources())
		// Before any migrations are applied, the status should be empty.
		status, err := p.Status(ctx)
		check.NoError(t, err)
		check.Number(t, len(status), numCount)
		assertStatus(t, status[0], goose.StatePending, newSource(goose.TypeSQL, "00001_users_table.sql", 1), true)
		assertStatus(t, status[1], goose.StatePending, newSource(goose.TypeSQL, "00002_posts_table.sql", 2), true)
		assertStatus(t, status[2], goose.StatePending, newSource(goose.TypeSQL, "00003_comments_table.sql", 3), true)
		assertStatus(t, status[3], goose.StatePending, newSource(goose.TypeSQL, "00004_insert_data.sql", 4), true)
		assertStatus(t, status[4], goose.StatePending, newSource(goose.TypeSQL, "00005_posts_view.sql", 5), true)
		assertStatus(t, status[5], goose.StatePending, newSource(goose.TypeSQL, "00006_empty_up.sql", 6), true)
		assertStatus(t, status[6], goose.StatePending, newSource(goose.TypeSQL, "00007_empty_up_down.sql", 7), true)
		// Apply all migrations
		_, err = p.Up(ctx)
		check.NoError(t, err)
		status, err = p.Status(ctx)
		check.NoError(t, err)
		check.Number(t, len(status), numCount)
		assertStatus(t, status[0], goose.StateApplied, newSource(goose.TypeSQL, "00001_users_table.sql", 1), false)
		assertStatus(t, status[1], goose.StateApplied, newSource(goose.TypeSQL, "00002_posts_table.sql", 2), false)
		assertStatus(t, status[2], goose.StateApplied, newSource(goose.TypeSQL, "00003_comments_table.sql", 3), false)
		assertStatus(t, status[3], goose.StateApplied, newSource(goose.TypeSQL, "00004_insert_data.sql", 4), false)
		assertStatus(t, status[4], goose.StateApplied, newSource(goose.TypeSQL, "00005_posts_view.sql", 5), false)
		assertStatus(t, status[5], goose.StateApplied, newSource(goose.TypeSQL, "00006_empty_up.sql", 6), false)
		assertStatus(t, status[6], goose.StateApplied, newSource(goose.TypeSQL, "00007_empty_up_down.sql", 7), false)
	})
	t.Run("tx_partial_errors", func(t *testing.T) {
		countOwners := func(db *sql.DB) (int, error) {
			q := `SELECT count(*)FROM owners`
			var count int
			if err := db.QueryRow(q).Scan(&count); err != nil {
				return 0, err
			}
			return count, nil
		}

		ctx := context.Background()
		db := newDB(t)
		mapFS := fstest.MapFS{
			"00001_users_table.sql": newMapFile(`
-- +goose Up
CREATE TABLE owners ( owner_name TEXT NOT NULL );
`),
			"00002_partial_error.sql": newMapFile(`
-- +goose Up
INSERT INTO invalid_table (invalid_table) VALUES ('invalid_value');
`),
			"00003_insert_data.sql": newMapFile(`
-- +goose Up
INSERT INTO owners (owner_name) VALUES ('seed-user-1');
INSERT INTO owners (owner_name) VALUES ('seed-user-2');
INSERT INTO owners (owner_name) VALUES ('seed-user-3');
`),
		}
		p, err := goose.NewProvider(goose.DialectSQLite3, db, mapFS)
		check.NoError(t, err)
		_, err = p.Up(ctx)
		check.HasError(t, err)
		check.Contains(t, err.Error(), "partial migration error (type:sql,version:2)")
		var expected *goose.PartialError
		check.Bool(t, errors.As(err, &expected), true)
		// Check Err field
		check.Bool(t, expected.Err != nil, true)
		check.Contains(t, expected.Err.Error(), "SQL logic error: no such table: invalid_table (1)")
		// Check Results field
		check.Number(t, len(expected.Applied), 1)
		assertResult(t, expected.Applied[0], newSource(goose.TypeSQL, "00001_users_table.sql", 1), "up", false)
		// Check Failed field
		check.Bool(t, expected.Failed != nil, true)
		assertSource(t, expected.Failed.Source, goose.TypeSQL, "00002_partial_error.sql", 2)
		check.Bool(t, expected.Failed.Empty, false)
		check.Bool(t, expected.Failed.Error != nil, true)
		check.Contains(t, expected.Failed.Error.Error(), "SQL logic error: no such table: invalid_table (1)")
		check.Equal(t, expected.Failed.Direction, "up")
		check.Bool(t, expected.Failed.Duration > 0, true)

		// Ensure the partial error did not affect the database.
		count, err := countOwners(db)
		check.NoError(t, err)
		check.Number(t, count, 0)

		status, err := p.Status(ctx)
		check.NoError(t, err)
		check.Number(t, len(status), 3)
		assertStatus(t, status[0], goose.StateApplied, newSource(goose.TypeSQL, "00001_users_table.sql", 1), false)
		assertStatus(t, status[1], goose.StatePending, newSource(goose.TypeSQL, "00002_partial_error.sql", 2), true)
		assertStatus(t, status[2], goose.StatePending, newSource(goose.TypeSQL, "00003_insert_data.sql", 3), true)
	})
}

func TestConcurrentProvider(t *testing.T) {
	t.Parallel()

	t.Run("up", func(t *testing.T) {
		ctx := context.Background()
		p, _ := newProviderWithDB(t)
		maxVersion := len(p.ListSources())

		ch := make(chan int64)
		var wg sync.WaitGroup
		for i := 0; i < maxVersion; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				res, err := p.UpByOne(ctx)
				if err != nil {
					t.Error(err)
					return
				}
				if res == nil {
					t.Errorf("expected non-nil result, got nil")
					return
				}
				ch <- res.Source.Version
			}()
		}
		go func() {
			wg.Wait()
			close(ch)
		}()
		var versions []int64
		for version := range ch {
			versions = append(versions, version)
		}
		// Fail early if any of the goroutines failed.
		if t.Failed() {
			return
		}
		check.Number(t, len(versions), maxVersion)
		for i := 0; i < maxVersion; i++ {
			check.Number(t, versions[i], int64(i+1))
		}
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, maxVersion)
	})
	t.Run("down", func(t *testing.T) {
		ctx := context.Background()
		p, _ := newProviderWithDB(t)
		maxVersion := len(p.ListSources())
		// Apply all migrations
		_, err := p.Up(ctx)
		check.NoError(t, err)
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, maxVersion)

		ch := make(chan []*goose.MigrationResult)
		var wg sync.WaitGroup
		for i := 0; i < maxVersion; i++ {
			wg.Add(1)

			go func() {
				defer wg.Done()
				res, err := p.DownTo(ctx, 0)
				if err != nil {
					t.Error(err)
					return
				}
				ch <- res
			}()
		}
		go func() {
			wg.Wait()
			close(ch)
		}()
		var (
			valid [][]*goose.MigrationResult
			empty [][]*goose.MigrationResult
		)
		for results := range ch {
			if len(results) == 0 {
				empty = append(empty, results)
				continue
			}
			valid = append(valid, results)
		}
		// Fail early if any of the goroutines failed.
		if t.Failed() {
			return
		}
		check.Equal(t, len(valid), 1)
		check.Equal(t, len(empty), maxVersion-1)
		// Ensure the valid result is correct.
		check.Number(t, len(valid[0]), maxVersion)
	})
}

func TestNoVersioning(t *testing.T) {
	t.Parallel()

	countSeedOwners := func(db *sql.DB) (int, error) {
		q := `SELECT count(*)FROM owners WHERE owner_name LIKE'seed-user-%'`
		var count int
		if err := db.QueryRow(q).Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	}
	countOwners := func(db *sql.DB) (int, error) {
		q := `SELECT count(*)FROM owners`
		var count int
		if err := db.QueryRow(q).Scan(&count); err != nil {
			return 0, err
		}
		return count, nil
	}
	ctx := context.Background()
	dbName := fmt.Sprintf("test_%s.db", randomAlphaNumeric(8))
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), dbName))
	check.NoError(t, err)
	fsys := os.DirFS(filepath.Join("testdata", "no-versioning", "migrations"))
	const (
		// Total owners created by the seed files.
		wantSeedOwnerCount = 250
		// These are owners created by migration files.
		wantOwnerCount = 4
	)
	p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys,
		goose.WithVerbose(testing.Verbose()),
		goose.WithDisableVersioning(false), // This is the default.
	)
	check.Number(t, len(p.ListSources()), 3)
	check.NoError(t, err)
	_, err = p.Up(ctx)
	check.NoError(t, err)
	baseVersion, err := p.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, baseVersion, 3)
	t.Run("seed-up-down-to-zero", func(t *testing.T) {
		fsys := os.DirFS(filepath.Join("testdata", "no-versioning", "seed"))
		p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys,
			goose.WithVerbose(testing.Verbose()),
			goose.WithDisableVersioning(true), // Provider with no versioning.
		)
		check.NoError(t, err)
		check.Number(t, len(p.ListSources()), 2)

		// Run (all) up migrations from the seed dir
		{
			upResult, err := p.Up(ctx)
			check.NoError(t, err)
			check.Number(t, len(upResult), 2)
			// When versioning is disabled, we cannot track the version of the seed files.
			_, err = p.GetDBVersion(ctx)
			check.HasError(t, err)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, wantSeedOwnerCount)
		}
		// Run (all) down migrations from the seed dir
		{
			downResult, err := p.DownTo(ctx, 0)
			check.NoError(t, err)
			check.Number(t, len(downResult), 2)
			// When versioning is disabled, we cannot track the version of the seed files.
			_, err = p.GetDBVersion(ctx)
			check.HasError(t, err)
			seedOwnerCount, err := countSeedOwners(db)
			check.NoError(t, err)
			check.Number(t, seedOwnerCount, 0)
		}
		// The migrations added 4 non-seed owners, they must remain in the database afterwards
		ownerCount, err := countOwners(db)
		check.NoError(t, err)
		check.Number(t, ownerCount, wantOwnerCount)
	})
}

func TestAllowMissing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Developer A and B check out the "main" branch which is currently on version 3. Developer A
	// mistakenly creates migration 5 and commits. Developer B did not pull the latest changes and
	// commits migration 4. Oops -- now the migrations are out of order.
	//
	// When goose is set to allow missing migrations, then 5 is applied after 4 with no error.
	// Otherwise it's expected to be an error.

	t.Run("missing_now_allowed", func(t *testing.T) {
		db := newDB(t)
		p, err := goose.NewProvider(goose.DialectSQLite3, db, newFsys(),
			goose.WithAllowOutofOrder(false),
		)
		check.NoError(t, err)

		// Create and apply first 3 migrations.
		_, err = p.UpTo(ctx, 3)
		check.NoError(t, err)
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 3)

		// Developer A - migration 5 (mistakenly applied)
		result, err := p.ApplyVersion(ctx, 5, true)
		check.NoError(t, err)
		check.Number(t, result.Source.Version, 5)
		current, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 5)

		// The database has migrations 1,2,3,5 applied.

		// Developer B is on version 3 (e.g., never pulled the latest changes). Adds migration 4. By
		// default goose does not allow missing (out-of-order) migrations, which means halt if a
		// missing migration is detected.
		_, err = p.Up(ctx)
		check.HasError(t, err)
		// found 1 missing (out-of-order) migration: [00004_insert_data.sql]
		check.Contains(t, err.Error(), "missing (out-of-order) migration")
		// Confirm db version is unchanged.
		current, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 5)

		_, err = p.UpByOne(ctx)
		check.HasError(t, err)
		// found 1 missing (out-of-order) migration: [00004_insert_data.sql]
		check.Contains(t, err.Error(), "missing (out-of-order) migration")
		// Confirm db version is unchanged.
		current, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 5)

		_, err = p.UpTo(ctx, math.MaxInt64)
		check.HasError(t, err)
		// found 1 missing (out-of-order) migration: [00004_insert_data.sql]
		check.Contains(t, err.Error(), "missing (out-of-order) migration")
		// Confirm db version is unchanged.
		current, err = p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, current, 5)
	})

	t.Run("missing_allowed", func(t *testing.T) {
		db := newDB(t)
		p, err := goose.NewProvider(goose.DialectSQLite3, db, newFsys(),
			goose.WithAllowOutofOrder(true),
		)
		check.NoError(t, err)

		// Create and apply first 3 migrations.
		_, err = p.UpTo(ctx, 3)
		check.NoError(t, err)
		currentVersion, err := p.GetDBVersion(ctx)
		check.NoError(t, err)
		check.Number(t, currentVersion, 3)

		// Developer A - migration 5 (mistakenly applied)
		{
			_, err = p.ApplyVersion(ctx, 5, true)
			check.NoError(t, err)
			current, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, current, 5)
		}
		// Developer B - migration 4 (missing) and 6 (new)
		{
			// 4
			upResult, err := p.UpByOne(ctx)
			check.NoError(t, err)
			check.Bool(t, upResult != nil, true)
			check.Number(t, upResult.Source.Version, 4)
			// 6
			upResult, err = p.UpByOne(ctx)
			check.NoError(t, err)
			check.Bool(t, upResult != nil, true)
			check.Number(t, upResult.Source.Version, 6)

			count, err := getGooseVersionCount(db, goose.DefaultTablename)
			check.NoError(t, err)
			check.Number(t, count, 6)
			current, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			// Expecting max(version_id) to be 8
			check.Number(t, current, 6)
		}

		// The applied order in the database is expected to be:
		//      1,2,3,5,4,6
		// So migrating down should be the reverse of the applied order:
		//      6,4,5,3,2,1

		testDownAndVersion := func(wantDBVersion, wantResultVersion int64) {
			currentVersion, err := p.GetDBVersion(ctx)
			check.NoError(t, err)
			check.Number(t, currentVersion, wantDBVersion)
			downRes, err := p.Down(ctx)
			check.NoError(t, err)
			check.Bool(t, downRes != nil, true)
			check.Number(t, downRes.Source.Version, wantResultVersion)
		}

		// This behaviour may need to change, see the following issues for more details:
		//  - https://github.com/pressly/goose/issues/523
		//  - https://github.com/pressly/goose/issues/402

		testDownAndVersion(6, 6)
		testDownAndVersion(5, 4) // Ensure the max db version is 5 before down.
		testDownAndVersion(5, 5)
		testDownAndVersion(3, 3)
		testDownAndVersion(2, 2)
		testDownAndVersion(1, 1)
		_, err = p.Down(ctx)
		check.HasError(t, err)
		check.Bool(t, errors.Is(err, goose.ErrNoNextVersion), true)
	})
}

func TestSQLiteSharedCache(t *testing.T) {
	t.Parallel()
	// goose uses *sql.Conn for most operations (incl. creating the initial table), but for Go
	// migrations when running outside a transaction we use *sql.DB. This is a problem for SQLite
	// because it does not support shared cache mode by default and it does not see the table that
	// was created through the initial connection. This test ensures goose works with SQLite shared
	// cache mode.
	//
	// Ref: https://www.sqlite.org/inmemorydb.html
	//
	// "In-memory databases are allowed to use shared cache if they are opened using a URI filename.
	// If the unadorned ":memory:" name is used to specify the in-memory database, then that
	// database always has a private cache and is only visible to the database connection that
	// originally opened it. However, the same in-memory database can be opened by two or more
	// database connections as follows: file::memory:?cache=shared"
	t.Run("shared_cache", func(t *testing.T) {
		db, err := sql.Open("sqlite", "file::memory:?cache=shared")
		check.NoError(t, err)
		fsys := fstest.MapFS{"00001_a.sql": newMapFile(`-- +goose Up`)}
		p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys,
			goose.WithGoMigrations(
				goose.NewGoMigration(2, &goose.GoFunc{Mode: goose.TransactionDisabled}, nil),
			),
		)
		check.NoError(t, err)
		_, err = p.Up(context.Background())
		check.NoError(t, err)
	})
	t.Run("no_shared_cache", func(t *testing.T) {
		db, err := sql.Open("sqlite", "file::memory:")
		check.NoError(t, err)
		fsys := fstest.MapFS{"00001_a.sql": newMapFile(`-- +goose Up`)}
		p, err := goose.NewProvider(goose.DialectSQLite3, db, fsys,
			goose.WithGoMigrations(
				goose.NewGoMigration(2, &goose.GoFunc{Mode: goose.TransactionDisabled}, nil),
			),
		)
		check.NoError(t, err)
		_, err = p.Up(context.Background())
		check.HasError(t, err)
		check.Contains(t, err.Error(), "SQL logic error: no such table: goose_db_version")
	})
}

func TestGoMigrationPanic(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	const (
		wantErrString = "panic: runtime error: index out of range [7] with length 0"
	)
	migration := goose.NewGoMigration(
		1,
		&goose.GoFunc{RunTx: func(ctx context.Context, tx *sql.Tx) error {
			var ss []int
			_ = ss[7]
			return nil
		}},
		nil,
	)
	p, err := goose.NewProvider(goose.DialectSQLite3, newDB(t), nil,
		goose.WithGoMigrations(migration), // Add a Go migration that panics.
	)
	check.NoError(t, err)
	_, err = p.Up(ctx)
	check.HasError(t, err)
	check.Contains(t, err.Error(), wantErrString)
	var expected *goose.PartialError
	check.Bool(t, errors.As(err, &expected), true)
	check.Contains(t, expected.Err.Error(), wantErrString)
}

func TestCustomStoreTableExists(t *testing.T) {
	t.Parallel()

	store, err := database.NewStore(database.DialectSQLite3, goose.DefaultTablename)
	check.NoError(t, err)
	p, err := goose.NewProvider("", newDB(t), newFsys(),
		goose.WithStore(&customStoreSQLite3{store}),
	)
	check.NoError(t, err)
	_, err = p.Up(context.Background())
	check.NoError(t, err)
}

func TestProviderApply(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	p, err := goose.NewProvider(goose.DialectSQLite3, newDB(t), newFsys())
	check.NoError(t, err)
	_, err = p.ApplyVersion(ctx, 1, true)
	check.NoError(t, err)
	// This version has a corresponding down migration, but has never been applied.
	_, err = p.ApplyVersion(ctx, 2, false)
	check.HasError(t, err)
	check.Bool(t, errors.Is(err, goose.ErrNotApplied), true)
}

func TestPending(t *testing.T) {
	t.Parallel()
	t.Run("allow_out_of_order", func(t *testing.T) {
		ctx := context.Background()
		fsys := newFsys()
		p, err := goose.NewProvider(goose.DialectSQLite3, newDB(t), fsys,
			goose.WithAllowOutofOrder(true),
		)
		check.NoError(t, err)
		// Some migrations have been applied out of order.
		_, err = p.ApplyVersion(ctx, 1, true)
		check.NoError(t, err)
		_, err = p.ApplyVersion(ctx, 3, true)
		check.NoError(t, err)
		// Even though the latest migration HAS been applied, there are still pending out-of-order
		// migrations.
		current, target, err := p.GetVersions(ctx)
		check.NoError(t, err)
		check.Number(t, current, 3)
		check.Number(t, target, len(fsys))
		hasPending, err := p.HasPending(ctx)
		check.NoError(t, err)
		check.Bool(t, hasPending, true)
		// Apply the missing migrations.
		_, err = p.Up(ctx)
		check.NoError(t, err)
		// All migrations have been applied.
		hasPending, err = p.HasPending(ctx)
		check.NoError(t, err)
		check.Bool(t, hasPending, false)
		current, target, err = p.GetVersions(ctx)
		check.NoError(t, err)
		check.Number(t, current, target)
	})
	t.Run("disallow_out_of_order", func(t *testing.T) {
		ctx := context.Background()
		fsys := newFsys()

		run := func(t *testing.T, versionToApply int64) {
			p, err := goose.NewProvider(goose.DialectSQLite3, newDB(t), fsys,
				goose.WithAllowOutofOrder(false),
			)
			check.NoError(t, err)
			// Some migrations have been applied.
			_, err = p.ApplyVersion(ctx, 1, true)
			check.NoError(t, err)
			_, err = p.ApplyVersion(ctx, versionToApply, true)
			check.NoError(t, err)
			// TODO(mf): revisit the pending check behavior in addition to the HasPending
			// method.
			current, target, err := p.GetVersions(ctx)
			check.NoError(t, err)
			check.Number(t, current, versionToApply)
			check.Number(t, target, len(fsys))
			_, err = p.HasPending(ctx)
			check.HasError(t, err)
			check.Contains(t, err.Error(), "missing (out-of-order) migration")
			_, err = p.Up(ctx)
			check.HasError(t, err)
			check.Contains(t, err.Error(), "missing (out-of-order) migration")
		}

		t.Run("latest_version", func(t *testing.T) {
			run(t, int64(len(fsys)))
		})
		t.Run("latest_version_minus_one", func(t *testing.T) {
			run(t, int64(len(fsys)-1))
		})
	})
}

type customStoreSQLite3 struct {
	database.Store
}

func (s *customStoreSQLite3) TableExists(ctx context.Context, db database.DBTxConn, name string) (bool, error) {
	q := `SELECT EXISTS (SELECT 1 FROM sqlite_master WHERE type='table' AND name=$1) AS table_exists`
	var exists bool
	if err := db.QueryRowContext(ctx, q, name).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func getGooseVersionCount(db *sql.DB, gooseTable string) (int64, error) {
	var gotVersion int64
	if err := db.QueryRow(
		fmt.Sprintf("SELECT count(*) FROM %s WHERE version_id > 0", gooseTable),
	).Scan(&gotVersion); err != nil {
		return 0, err
	}
	return gotVersion, nil
}

func TestGoOnly(t *testing.T) {
	t.Cleanup(goose.ResetGlobalMigrations)
	// Not parallel because each subtest modifies global state.

	countUser := func(db *sql.DB) int {
		q := `SELECT count(*)FROM users`
		var count int
		err := db.QueryRow(q).Scan(&count)
		check.NoError(t, err)
		return count
	}

	t.Run("with_tx", func(t *testing.T) {
		ctx := context.Background()
		register := []*goose.Migration{
			goose.NewGoMigration(
				1,
				&goose.GoFunc{RunTx: newTxFn("CREATE TABLE users (id INTEGER PRIMARY KEY)")},
				&goose.GoFunc{RunTx: newTxFn("DROP TABLE users")},
			),
		}
		err := goose.SetGlobalMigrations(register...)
		check.NoError(t, err)
		t.Cleanup(goose.ResetGlobalMigrations)

		db := newDB(t)
		register = []*goose.Migration{
			goose.NewGoMigration(
				2,
				&goose.GoFunc{RunTx: newTxFn("INSERT INTO users (id) VALUES (1), (2), (3)")},
				&goose.GoFunc{RunTx: newTxFn("DELETE FROM users")},
			),
		}
		p, err := goose.NewProvider(goose.DialectSQLite3, db, nil,
			goose.WithGoMigrations(register...),
		)
		check.NoError(t, err)
		sources := p.ListSources()
		check.Number(t, len(p.ListSources()), 2)
		assertSource(t, sources[0], goose.TypeGo, "", 1)
		assertSource(t, sources[1], goose.TypeGo, "", 2)
		// Apply migration 1
		res, err := p.UpByOne(ctx)
		check.NoError(t, err)
		assertResult(t, res, newSource(goose.TypeGo, "", 1), "up", false)
		check.Number(t, countUser(db), 0)
		check.Bool(t, tableExists(t, db, "users"), true)
		// Apply migration 2
		res, err = p.UpByOne(ctx)
		check.NoError(t, err)
		assertResult(t, res, newSource(goose.TypeGo, "", 2), "up", false)
		check.Number(t, countUser(db), 3)
		// Rollback migration 2
		res, err = p.Down(ctx)
		check.NoError(t, err)
		assertResult(t, res, newSource(goose.TypeGo, "", 2), "down", false)
		check.Number(t, countUser(db), 0)
		// Rollback migration 1
		res, err = p.Down(ctx)
		check.NoError(t, err)
		assertResult(t, res, newSource(goose.TypeGo, "", 1), "down", false)
		// Check table does not exist
		check.Bool(t, tableExists(t, db, "users"), false)
	})
	t.Run("with_db", func(t *testing.T) {
		ctx := context.Background()
		register := []*goose.Migration{
			goose.NewGoMigration(
				1,
				&goose.GoFunc{
					RunDB: newDBFn("CREATE TABLE users (id INTEGER PRIMARY KEY)"),
				},
				&goose.GoFunc{
					RunDB: newDBFn("DROP TABLE users"),
				},
			),
		}
		err := goose.SetGlobalMigrations(register...)
		check.NoError(t, err)
		t.Cleanup(goose.ResetGlobalMigrations)

		db := newDB(t)
		register = []*goose.Migration{
			goose.NewGoMigration(
				2,
				&goose.GoFunc{RunDB: newDBFn("INSERT INTO users (id) VALUES (1), (2), (3)")},
				&goose.GoFunc{RunDB: newDBFn("DELETE FROM users")},
			),
		}
		p, err := goose.NewProvider(goose.DialectSQLite3, db, nil,
			goose.WithGoMigrations(register...),
		)
		check.NoError(t, err)
		sources := p.ListSources()
		check.Number(t, len(p.ListSources()), 2)
		assertSource(t, sources[0], goose.TypeGo, "", 1)
		assertSource(t, sources[1], goose.TypeGo, "", 2)
		// Apply migration 1
		res, err := p.UpByOne(ctx)
		check.NoError(t, err)
		assertResult(t, res, newSource(goose.TypeGo, "", 1), "up", false)
		check.Number(t, countUser(db), 0)
		check.Bool(t, tableExists(t, db, "users"), true)
		// Apply migration 2
		res, err = p.UpByOne(ctx)
		check.NoError(t, err)
		assertResult(t, res, newSource(goose.TypeGo, "", 2), "up", false)
		check.Number(t, countUser(db), 3)
		// Rollback migration 2
		res, err = p.Down(ctx)
		check.NoError(t, err)
		assertResult(t, res, newSource(goose.TypeGo, "", 2), "down", false)
		check.Number(t, countUser(db), 0)
		// Rollback migration 1
		res, err = p.Down(ctx)
		check.NoError(t, err)
		assertResult(t, res, newSource(goose.TypeGo, "", 1), "down", false)
		// Check table does not exist
		check.Bool(t, tableExists(t, db, "users"), false)
	})
}

func newDBFn(query string) func(context.Context, *sql.DB) error {
	return func(ctx context.Context, db *sql.DB) error {
		_, err := db.ExecContext(ctx, query)
		return err
	}
}

func newTxFn(query string) func(context.Context, *sql.Tx) error {
	return func(ctx context.Context, tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query)
		return err
	}
}

func tableExists(t *testing.T, db *sql.DB, table string) bool {
	q := fmt.Sprintf(`SELECT CASE WHEN COUNT(*) > 0 THEN 1 ELSE 0 END AS table_exists FROM sqlite_master WHERE type = 'table' AND name = '%s'`, table)
	var b string
	err := db.QueryRow(q).Scan(&b)
	check.NoError(t, err)
	return b == "1"
}

const (
	charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

func randomAlphaNumeric(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

func newProviderWithDB(t *testing.T, opts ...goose.ProviderOption) (*goose.Provider, *sql.DB) {
	t.Helper()
	db := newDB(t)
	opts = append(
		opts,
		goose.WithVerbose(testing.Verbose()),
	)
	p, err := goose.NewProvider(goose.DialectSQLite3, db, newFsys(), opts...)
	check.NoError(t, err)
	return p, db
}

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	dbName := fmt.Sprintf("test_%s.db", randomAlphaNumeric(8))
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), dbName))
	check.NoError(t, err)
	return db
}

func getMaxVersionID(db *sql.DB, gooseTable string) (int64, error) {
	var gotVersion int64
	if err := db.QueryRow(
		fmt.Sprintf("select max(version_id) from %s", gooseTable),
	).Scan(&gotVersion); err != nil {
		return 0, err
	}
	return gotVersion, nil
}

func getTableNames(db *sql.DB) ([]string, error) {
	rows, err := db.Query(`SELECT name FROM sqlite_master WHERE type='table' ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func assertStatus(t *testing.T, got *goose.MigrationStatus, state goose.State, source *goose.Source, appliedIsZero bool) {
	t.Helper()
	check.Equal(t, got.State, state)
	check.Equal(t, got.Source, source)
	check.Bool(t, got.AppliedAt.IsZero(), appliedIsZero)
}

func assertResult(t *testing.T, got *goose.MigrationResult, source *goose.Source, direction string, isEmpty bool) {
	t.Helper()
	check.Bool(t, got != nil, true)
	check.Equal(t, got.Source, source)
	check.Equal(t, got.Direction, direction)
	check.Equal(t, got.Empty, isEmpty)
	check.Bool(t, got.Error == nil, true)
	check.Bool(t, got.Duration > 0, true)
}

func assertSource(t *testing.T, got *goose.Source, typ goose.MigrationType, name string, version int64) {
	t.Helper()
	check.Equal(t, got.Type, typ)
	check.Equal(t, got.Path, name)
	check.Equal(t, got.Version, version)
}

func newSource(t goose.MigrationType, fullpath string, version int64) *goose.Source {
	return &goose.Source{
		Type:    t,
		Path:    fullpath,
		Version: version,
	}
}

func newMapFile(data string) *fstest.MapFile {
	return &fstest.MapFile{
		Data: []byte(data),
	}
}

func newFsys() fstest.MapFS {
	return fstest.MapFS{
		"00001_users_table.sql":    newMapFile(runMigration1),
		"00002_posts_table.sql":    newMapFile(runMigration2),
		"00003_comments_table.sql": newMapFile(runMigration3),
		"00004_insert_data.sql":    newMapFile(runMigration4),
		"00005_posts_view.sql":     newMapFile(runMigration5),
		"00006_empty_up.sql":       newMapFile(runMigration6),
		"00007_empty_up_down.sql":  newMapFile(runMigration7),
	}
}

var (

	// known tables are the tables (including goose table) created by running all migration files.
	// If you add a table, make sure to add to this list and keep it in order.
	knownTables = []string{
		"comments",
		"goose_db_version",
		"posts",
		"sqlite_sequence",
		"users",
	}

	runMigration1 = `
-- +goose Up
CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL,
    email TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
DROP TABLE users;
`

	runMigration2 = `
-- +goose Up
-- +goose StatementBegin
CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    author_id INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (author_id) REFERENCES users(id)
);
-- +goose StatementEnd
SELECT 1;
SELECT 2;

-- +goose Down
DROP TABLE posts;
`

	runMigration3 = `
-- +goose Up
CREATE TABLE comments (
    id INTEGER PRIMARY KEY,
    post_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_id) REFERENCES posts(id),
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- +goose Down
DROP TABLE comments;
SELECT 1;
SELECT 2;
SELECT 3;
`

	runMigration4 = `
-- +goose Up
INSERT INTO users (id, username, email)
VALUES
    (1, 'john_doe', 'john@example.com'),
    (2, 'jane_smith', 'jane@example.com'),
    (3, 'alice_wonderland', 'alice@example.com');

INSERT INTO posts (id, title, content, author_id)
VALUES
    (1, 'Introduction to SQL', 'SQL is a powerful language for managing databases...', 1),
    (2, 'Data Modeling Techniques', 'Choosing the right data model is crucial...', 2),
    (3, 'Advanced Query Optimization', 'Optimizing queries can greatly improve...', 1);

INSERT INTO comments (id, post_id, user_id, content)
VALUES
    (1, 1, 3, 'Great introduction! Looking forward to more.'),
    (2, 1, 2, 'SQL can be a bit tricky at first, but practice helps.'),
    (3, 2, 1, 'You covered normalization really well in this post.');

-- +goose Down
DELETE FROM comments;
DELETE FROM posts;
DELETE FROM users;
`

	runMigration5 = `
-- +goose NO TRANSACTION

-- +goose Up
CREATE VIEW posts_view AS
    SELECT
        p.id,
        p.title,
        p.content,
        p.created_at,
        u.username AS author
    FROM posts p
    JOIN users u ON p.author_id = u.id;

-- +goose Down
DROP VIEW posts_view;
`

	runMigration6 = `
-- +goose Up
`

	runMigration7 = `
-- +goose Up
-- +goose Down
`
)
