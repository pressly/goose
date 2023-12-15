package vertica_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/testdb"
)

/*
This test applies all up migrations, asserts we have all the entries in
the versions table, applies all down migration and asserts we have zero
migrations applied.

Limitations:
1) Only one instance of Vertica can be running on a host at any time.
*/
func TestVerticaUpDownAll(t *testing.T) {
	t.Parallel()

	migrationDir := filepath.Join("testdata", "migrations")
	db, cleanup, err := testdb.NewVertica()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	check.NoError(t, goose.SetDialect("vertica"))

	goose.SetTableName("goose_db_version")

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

	type result struct {
		TestKey    int64     `db:"test_key"`
		TestID     string    `db:"test_id"`
		ValidFrom  time.Time `db:"valid_from"`
		ValidTo    time.Time `db:"valid_to"`
		IsCurrent  bool      `db:"is_current"`
		ExternalID string    `db:"external_id"`
	}
	rows, err := db.Query(`SELECT * FROM testing.dim_test_scd ORDER BY test_key`)
	check.NoError(t, err)
	var results []result
	for rows.Next() {
		var r result
		err = rows.Scan(&r.TestKey, &r.TestID, &r.ValidFrom, &r.ValidTo, &r.IsCurrent, &r.ExternalID)
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
		check.Equal(t, result.TestKey, want[i].TestKey)
		check.Equal(t, result.TestID, want[i].TestID)
		check.Equal(t, result.ValidFrom, want[i].ValidFrom)
		check.Equal(t, result.ValidTo, want[i].ValidTo)
		check.Equal(t, result.IsCurrent, want[i].IsCurrent)
		check.Equal(t, result.ExternalID, want[i].ExternalID)
	}

	err = goose.DownTo(db, migrationDir, 0)
	check.NoError(t, err)
	check.NoError(t, err)

	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)
}
