package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/testing/testdb"
	"github.com/stretchr/testify/require"
)

func TestPostgres(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewPostgres()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectPostgres, db, "testdata/migrations/postgres")
}

func TestSpanner(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewSpanner()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectSpanner, db, "testdata/migrations/spanner", goose.WithIsolateDDL(true))
}

func TestClickhouse(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewClickHouse()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectClickHouse, db, "testdata/migrations/clickhouse")

	type result struct {
		customerID     string    `db:"customer_id"`
		timestamp      time.Time `db:"time_stamp"`
		clickEventType string    `db:"click_event_type"`
		countryCode    string    `db:"country_code"`
		sourceID       int64     `db:"source_id"`
	}
	rows, err := db.Query(`SELECT * FROM clickstream ORDER BY customer_id`)
	require.NoError(t, err)
	var results []result
	for rows.Next() {
		var r result
		err = rows.Scan(&r.customerID, &r.timestamp, &r.clickEventType, &r.countryCode, &r.sourceID)
		require.NoError(t, err)
		results = append(results, r)
	}
	require.Equal(t, len(results), 3)
	require.NoError(t, rows.Close())
	require.NoError(t, rows.Err())

	parseTime := func(t *testing.T, s string) time.Time {
		t.Helper()
		tm, err := time.Parse("2006-01-02", s)
		require.NoError(t, err)
		return tm
	}
	want := []result{
		{"customer1", parseTime(t, "2021-10-02"), "add_to_cart", "US", 568239},
		{"customer2", parseTime(t, "2021-10-30"), "remove_from_cart", "", 0},
		{"customer3", parseTime(t, "2021-11-07"), "checkout", "", 307493},
	}
	for i, result := range results {
		require.Equal(t, result.customerID, want[i].customerID)
		require.Equal(t, result.timestamp, want[i].timestamp)
		require.Equal(t, result.clickEventType, want[i].clickEventType)
		if result.countryCode != "" && want[i].countryCode != "" {
			require.Equal(t, result.countryCode, want[i].countryCode)
		}
		require.Equal(t, result.sourceID, want[i].sourceID)
	}
}

func TestClickhouseReplicated(t *testing.T) {
	// Not t.Parallel(): the helper picks ephemeral host ports so multiple
	// runs *could* coexist, but the cluster startup is heavyweight (~10s),
	// so keeping this serial matches how the other integration tests behave.
	const cluster = "goose_cluster"
	t.Setenv(database.EnvClickhouseCluster, cluster)

	db, ch2, cleanup, err := testdb.NewClickHouseReplicated()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())
	require.NoError(t, ch2.Ping())

	testDatabase(t, database.DialectClickHouseReplicated, db, "testdata/migrations/clickhouse-replicated", goose.WithIsolateDDL(true))

	// testDatabase() leaves every migration applied. Roll one back and verify
	// that goose treats it as not-applied everywhere it checks version state,
	// not just via ListMigrations. This dialect records down-migrations as a
	// tombstone insert (is_applied = 0) rather than deleting the row, so
	// GetMigrationByVersion must independently agree that a tombstoned
	// version is not currently applied -- Provider.status and
	// Provider.ApplyVersion go through GetMigration, not ListMigrations, and
	// previously kept reporting a rolled-back version as applied because
	// GetMigrationByVersion returned its (stale) row regardless of is_applied.
	ctx := context.Background()
	p, err := goose.NewProvider(
		database.DialectClickHouseReplicated, db, os.DirFS("testdata/migrations/clickhouse-replicated"),
		goose.WithIsolateDDL(true),
	)
	require.NoError(t, err)

	downResult, err := p.Down(ctx)
	require.NoError(t, err)
	rolledBack := downResult.Source.Version

	status, err := p.Status(ctx)
	require.NoError(t, err)
	var rolledBackStatus *goose.MigrationStatus
	for _, s := range status {
		if s.Source.Version == rolledBack {
			rolledBackStatus = s
		}
	}
	require.NotNil(t, rolledBackStatus, "rolled-back version missing from status")
	require.Equal(t, goose.StatePending, rolledBackStatus.State,
		"rolled-back version must not report as applied")

	// Rolling it back again must fail with ErrNotApplied, not silently
	// re-tombstone an already-tombstoned version.
	_, err = p.ApplyVersion(ctx, rolledBack, false)
	require.ErrorIs(t, err, goose.ErrNotApplied)

	// Re-applying it must succeed -- must not be rejected as ErrAlreadyApplied.
	_, err = p.ApplyVersion(ctx, rolledBack, true)
	require.NoError(t, err)

	// Applying it a second time must now correctly fail as already applied.
	_, err = p.ApplyVersion(ctx, rolledBack, true)
	require.ErrorIs(t, err, goose.ErrAlreadyApplied)

	// Forward, then down, then forward to a *different* target than before:
	// roll back two versions (3, 2), re-apply only one of them (2), and check
	// that the still-tombstoned version (3) is not resurrected as applied by
	// the presence of its own earlier apply-row in the same table -- each
	// version's state must be judged independently via argMax, not smeared
	// across versions.
	//
	// This also leaves the table holding a row for version 3 (tombstoned)
	// whose version_id is *higher* than the resulting current version (2).
	// GetLatestVersion and ListMigrations must both filter to is_applied = 1
	// before taking max()/ORDER BY version_id, or that stale high-numbered
	// row would leak through and report the wrong current version.
	downTo, err := p.DownTo(ctx, 1)
	require.NoError(t, err)
	require.Len(t, downTo, 2, "expected versions 3 and 2 to be rolled back")

	version, err := p.GetDBVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), version)

	upTo, err := p.UpTo(ctx, 2)
	require.NoError(t, err)
	require.Len(t, upTo, 1, "expected only version 2 to be re-applied")
	require.Equal(t, int64(2), upTo[0].Source.Version)

	version, err = p.GetDBVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), version,
		"current version must be 2 even though a row for version 3 still exists in the table")

	status, err = p.Status(ctx)
	require.NoError(t, err)
	for _, s := range status {
		switch s.Source.Version {
		case 1, 2:
			require.Equal(t, goose.StateApplied, s.State, "version %d", s.Source.Version)
		case 3:
			require.Equal(t, goose.StatePending, s.State, "version 3 must not be reported applied")
		}
	}

	// Restore the fully-applied state expected by the replication checks below.
	upTo, err = p.UpTo(ctx, 3)
	require.NoError(t, err)
	require.Len(t, upTo, 1, "expected only version 3 to be re-applied")
	version, err = p.GetDBVersion(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(3), version)

	// Regression test for the argMax tie-break (commit fb3ca3b): when an
	// apply-row and a tombstone-row for the same version_id land on the
	// *identical* tstamp -- possible under concurrent, uninterlocked writers,
	// which this dialect otherwise assumes are serialized externally -- the
	// tombstone must still win, and must win regardless of which of the two
	// rows was physically inserted first. Manipulate the version table
	// directly since goose itself never produces two rows with the same
	// tstamp.
	store, err := database.NewStore(database.DialectClickHouseReplicated, goose.DefaultTablename)
	require.NoError(t, err)
	insertVersionRow := func(version int64, isApplied int, ts time.Time) {
		q := fmt.Sprintf(
			`INSERT INTO %s (version_id, is_applied, tstamp) SETTINGS insert_quorum='auto' VALUES (%d, %d, toDateTime64('%s', 6))`,
			goose.DefaultTablename, version, isApplied, ts.UTC().Format("2006-01-02 15:04:05.999999"),
		)
		_, err := db.ExecContext(ctx, q)
		require.NoError(t, err)
	}
	assertTombstoneWinsTie := func(t *testing.T, version int64) {
		t.Helper()
		_, err := store.GetMigration(ctx, db, version)
		require.ErrorIs(t, err, database.ErrVersionNotFound,
			"tombstone must win a tstamp tie in GetMigration")

		latest, err := store.GetLatestVersion(ctx, db)
		require.NoError(t, err)
		require.NotEqual(t, version, latest, "tied version must not win GetLatestVersion")

		listed, err := store.ListMigrations(ctx, db)
		require.NoError(t, err)
		for _, m := range listed {
			require.NotEqual(t, version, m.Version, "tied version must not appear in ListMigrations")
		}
	}

	const tieVersionApplyFirst = 900001
	tieTS := time.Now()
	insertVersionRow(tieVersionApplyFirst, 1, tieTS)
	insertVersionRow(tieVersionApplyFirst, 0, tieTS)
	assertTombstoneWinsTie(t, tieVersionApplyFirst)

	const tieVersionTombstoneFirst = 900002
	insertVersionRow(tieVersionTombstoneFirst, 0, tieTS)
	insertVersionRow(tieVersionTombstoneFirst, 1, tieTS)
	assertTombstoneWinsTie(t, tieVersionTombstoneFirst)

	// After testDatabase() completes, all up-migrations have been re-applied
	// (UpByOne loop at the end). Verify that the seeded rows and the
	// goose_db_version bookkeeping are visible on BOTH replicas. Replication
	// itself is a property of ClickHouse's Replicated* engines, not of the
	// dialect; what this checks is that the dialect drives them correctly
	// (ON CLUSTER DDL on both nodes, replicated version table, replicated
	// user tables via the migrations under testdata/) so the version state
	// actually converges on ch2.
	//
	// Replication is asynchronous by default; select_sequential_consistency
	// wouldn't help here because we're bypassing the dialect's Querier for a
	// raw SELECT.
	require.Eventually(t, func() bool {
		var got int
		if err := ch2.QueryRow(`SELECT count() FROM events`).Scan(&got); err != nil {
			return false
		}
		return got == 3
	}, 30*time.Second, 500*time.Millisecond, "expected 3 rows to replicate to ch2")

	require.Eventually(t, func() bool {
		var got int
		if err := ch2.QueryRow(`SELECT count() FROM (
			SELECT version_id, argMax(is_applied, tstamp) AS is_applied
			FROM goose_db_version GROUP BY version_id
		) WHERE version_id > 0 AND is_applied = 1`).Scan(&got); err != nil {
			return false
		}
		return got == 3
	}, 30*time.Second, 500*time.Millisecond, "expected 3 applied versions to replicate to ch2")
}

// TestClickhouseReplicated_EngineMismatch verifies that pointing one of the clickhouse /
// clickhouse-replicated dialects at a version table actually created by the other -- via the
// Provider API -- fails immediately with a clear error instead of silently reusing the
// incompatible table. See the TableExists implementations in internal/dialects/clickhouse.go and
// clickhouse_replicated.go.
func TestClickhouseReplicated_EngineMismatch(t *testing.T) {
	const cluster = "goose_cluster"
	t.Setenv(database.EnvClickhouseCluster, cluster)

	db, ch2, cleanup, err := testdb.NewClickHouseReplicated()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())
	require.NoError(t, ch2.Ping())

	ctx := context.Background()
	const migrationsDir = "testdata/migrations/clickhouse-replicated"

	// Create the version table via clickhouse-replicated first.
	replicatedProvider, err := goose.NewProvider(
		database.DialectClickHouseReplicated, db, os.DirFS(migrationsDir),
		goose.WithIsolateDDL(true),
	)
	require.NoError(t, err)
	_, err = replicatedProvider.Up(ctx)
	require.NoError(t, err)

	// Now point the stock clickhouse dialect at the same (already-created) table. The mismatch
	// must be caught before any migration runs, so it doesn't matter that these migrations were
	// written for the replicated dialect (e.g. ON CLUSTER DDL).
	stockProvider, err := goose.NewProvider(database.DialectClickHouse, db, os.DirFS(migrationsDir))
	require.NoError(t, err)
	_, err = stockProvider.Up(ctx)
	require.Error(t, err)
	require.ErrorContains(t, err, "incompatible engine")
	require.ErrorContains(t, err, "clickhouse-replicated")
}

func TestClickhouseRemote(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewClickHouse()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())
	testDatabase(t, database.DialectClickHouse, db, "testdata/migrations/clickhouse-remote")

	// assert that the taxi_zone_dictionary table has been created and populated
	var count int
	err = db.QueryRow(`SELECT count(*) FROM taxi_zone_dictionary`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 265, count)
}

func TestMySQL(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewMariaDB()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectMySQL, db, "testdata/migrations/mysql")
}

func TestTurso(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewTurso()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectTurso, db, "testdata/migrations/turso")
}

func TestYDB(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewYdb()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectYdB, db, "testdata/migrations/ydb")
}

func TestStarrocks(t *testing.T) {
	t.Parallel()

	// t.Skip("Starrocks is flaky on CI, see https://github.com/pressly/goose/issues/881")

	db, cleanup, err := testdb.NewStarrocks()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectStarrocks, db, "testdata/migrations/starrocks", goose.WithIsolateDDL(true))
}
