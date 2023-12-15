package clickhouse_test

import (
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/testdb"
)

func TestMain(m *testing.M) {
	if err := goose.SetDialect("clickhouse"); err != nil {
		log.Fatal(err)
	}
	os.Exit(m.Run())
}

func TestClickUpDownAll(t *testing.T) {
	t.Parallel()

	migrationDir := filepath.Join("testdata", "migrations")
	db, cleanup, err := testdb.NewClickHouse()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	/*
		This test applies all up migrations, asserts we have all the entries in
		the versions table, applies all down migration and asserts we have zero
		migrations applied.

		ClickHouse performs UPDATES and DELETES asynchronously,
		but we can best-effort check mutations and their progress.

		This is especially important for down migrations where rows are deleted
		from the versions table.

		For the sake of testing, there might be a way to modifying the server
		(or queries) to perform all operations synchronously?

		Ref: https://clickhouse.com/docs/en/operations/system-tables/mutations/
		Ref: https://clickhouse.com/docs/en/sql-reference/statements/alter/#mutations
		Ref: https://clickhouse.com/blog/how-to-update-data-in-click-house/
	*/

	// Collect migrations so we don't have to hard-code the currentVersion
	// in an assertion later in the test.
	migrations, err := goose.CollectMigrations(migrationDir, 0, goose.MaxVersion)
	check.NoError(t, err)

	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)

	err = goose.Up(db, migrationDir)
	check.NoError(t, err)
	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, len(migrations))

	err = goose.DownTo(db, migrationDir, 0)
	check.NoError(t, err)

	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)
}

func TestClickHouseFirstThree(t *testing.T) {
	t.Parallel()

	migrationDir := filepath.Join("testdata", "migrations")
	db, cleanup, err := testdb.NewClickHouse()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	err = goose.Up(db, migrationDir)
	check.NoError(t, err)

	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 3)

	type result struct {
		customerID     string    `db:"customer_id"`
		timestamp      time.Time `db:"time_stamp"`
		clickEventType string    `db:"click_event_type"`
		countryCode    string    `db:"country_code"`
		sourceID       int64     `db:"source_id"`
	}
	rows, err := db.Query(`SELECT * FROM clickstream ORDER BY customer_id`)
	check.NoError(t, err)
	var results []result
	for rows.Next() {
		var r result
		err = rows.Scan(&r.customerID, &r.timestamp, &r.clickEventType, &r.countryCode, &r.sourceID)
		check.NoError(t, err)
		results = append(results, r)
	}
	check.Number(t, len(results), 3)
	check.NoError(t, rows.Close())
	check.NoError(t, rows.Err())

	parseTime := func(t *testing.T, s string) time.Time {
		t.Helper()
		tm, err := time.Parse("2006-01-02", s)
		check.NoError(t, err)
		return tm
	}
	want := []result{
		{"customer1", parseTime(t, "2021-10-02"), "add_to_cart", "US", 568239},
		{"customer2", parseTime(t, "2021-10-30"), "remove_from_cart", "", 0},
		{"customer3", parseTime(t, "2021-11-07"), "checkout", "", 307493},
	}
	for i, result := range results {
		check.Equal(t, result.customerID, want[i].customerID)
		check.Equal(t, result.timestamp, want[i].timestamp)
		check.Equal(t, result.clickEventType, want[i].clickEventType)
		if result.countryCode != "" && want[i].countryCode != "" {
			check.Equal(t, result.countryCode, want[i].countryCode)
		}
		check.Number(t, result.sourceID, want[i].sourceID)
	}
}

func TestRemoteImportMigration(t *testing.T) {
	t.Parallel()
	// TODO(mf): use TestMain and create a proper "long" or "remote" flag.
	if !testing.Short() {
		t.Skip("skipping test")
	}
	// This test is using a remote dataset from an s3 bucket:
	// https://datasets-documentation.s3.eu-west-3.amazonaws.com/nyc-taxi/taxi_zone_lookup.csv
	// From this tutorial: https://clickhouse.com/docs/en/tutorial/
	// Note, these files are backed up in this repository in:
	// 		tests/clickhouse/testdata/backup-files/taxi_zone_lookup.csv
	// We may want to host this ourselves. Or just don't bother with SOURCE(HTTP(URL..
	// and craft a long INSERT statement.

	migrationDir := filepath.Join("testdata", "migrations-remote")
	db, cleanup, err := testdb.NewClickHouse()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	err = goose.Up(db, migrationDir)
	check.NoError(t, err)
	_, err = goose.GetDBVersion(db)
	check.NoError(t, err)

	var count int
	err = db.QueryRow(`SELECT count(*) FROM taxi_zone_dictionary`).Scan(&count)
	check.NoError(t, err)
	check.Number(t, count, 265)
}
