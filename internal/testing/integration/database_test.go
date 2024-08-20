package integration

import (
	"testing"
	"time"

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

func TestVertica(t *testing.T) {
	t.Parallel()

	db, cleanup, err := testdb.NewVertica()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectVertica, db, "testdata/migrations/vertica")

	type result struct {
		TestKey    int64     `db:"test_key"`
		TestID     string    `db:"test_id"`
		ValidFrom  time.Time `db:"valid_from"`
		ValidTo    time.Time `db:"valid_to"`
		IsCurrent  bool      `db:"is_current"`
		ExternalID string    `db:"external_id"`
	}
	rows, err := db.Query(`SELECT * FROM testing.dim_test_scd ORDER BY test_key`)
	require.NoError(t, err)
	var results []result
	for rows.Next() {
		var r result
		err = rows.Scan(&r.TestKey, &r.TestID, &r.ValidFrom, &r.ValidTo, &r.IsCurrent, &r.ExternalID)
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
		{
			TestKey:    1,
			TestID:     "575a0dd4-bd97-44ac-aae0-987090181da8",
			ValidFrom:  parseTime(t, "2021-10-02"),
			ValidTo:    parseTime(t, "2021-10-03"),
			IsCurrent:  false,
			ExternalID: "123",
		},
		{
			TestKey:    2,
			TestID:     "575a0dd4-bd97-44ac-aae0-987090181da8",
			ValidFrom:  parseTime(t, "2021-10-03"),
			ValidTo:    parseTime(t, "2021-10-04"),
			IsCurrent:  false,
			ExternalID: "456",
		},
		{
			TestKey:    3,
			TestID:     "575a0dd4-bd97-44ac-aae0-987090181da8",
			ValidFrom:  parseTime(t, "2021-10-04"),
			ValidTo:    parseTime(t, "9999-12-31"),
			IsCurrent:  true,
			ExternalID: "789",
		},
	}
	for i, result := range results {
		require.Equal(t, result.TestKey, want[i].TestKey)
		require.Equal(t, result.TestID, want[i].TestID)
		require.Equal(t, result.ValidFrom, want[i].ValidFrom)
		require.Equal(t, result.ValidTo, want[i].ValidTo)
		require.Equal(t, result.IsCurrent, want[i].IsCurrent)
		require.Equal(t, result.ExternalID, want[i].ExternalID)
	}
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

	db, cleanup, err := testdb.NewStarrocks()
	require.NoError(t, err)
	t.Cleanup(cleanup)
	require.NoError(t, db.Ping())

	testDatabase(t, database.DialectStarrocks, db, "testdata/migrations/starrocks")
}
