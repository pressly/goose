package clickhouse_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/testdb"
)

func TestClickUpDownAll(t *testing.T) {
	t.Parallel()

	migrationDir := filepath.Join("testdata", "migrations")
	db, cleanup, err := testdb.NewClickHouse()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	goose.SetDialect("clickhouse")

	retryCheckTableMutation := func(table string) func() error {
		return func() error {
			ok := checkTableMutation(t, db, table)
			if !ok {
				return errors.New("mutation not done for table: " + table)
			}
			return nil
		}
	}

	/*
		This test applies all up migrations, asserts we have all the entries in
		the versions table, applies all down migration and asserts we have 0
		migration versions.

		ClickHouse performs UPDATES and DELETES asynchronously,
		but we can best-effort check mutations and their progress retrying.

		This is especially important for down migrations where we delete rows.

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
	err = retry.Do(
		retryCheckTableMutation(goose.TableName()),
		retry.Delay(1*time.Second),
	)
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

	goose.SetDialect("clickhouse")

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

func checkTableMutation(t *testing.T, db *sql.DB, tableName string) bool {
	t.Helper()
	rows, err := db.Query(
		`select mutation_id, command, is_done, create_time from system.mutations where table=$1`,
		tableName,
	)
	check.NoError(t, err)

	type result struct {
		mutationID string    `db:"mutation_id"`
		command    string    `db:"command"`
		isDone     int64     `db:"is_done"`
		createTime time.Time `db:"create_time"`
	}
	var results []result
	for rows.Next() {
		var r result
		err = rows.Scan(&r.mutationID, &r.command, &r.isDone, &r.createTime)
		check.NoError(t, err)
		results = append(results, r)
	}
	check.NoError(t, rows.Close())
	check.NoError(t, rows.Err())
	if len(results) == 0 {
		return true
	}
	// Loop through all the mutations, if at least one of them is
	// not done, return false.
	for _, r := range results {
		if r.isDone != 1 {
			return false
		}
	}
	return true
}
