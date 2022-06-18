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

func TestClickHouseUpDownOneMigration(t *testing.T) {
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
		This test applies 1 up migration, asserts we have 1 entry in the
		versions table, applies 1 down migration and asserts we have 0
		versions.

		ClickHouse performs UPDATES and DELETES asynchronously,
		but we can best-effort check mutations and their progress retrying.

		This is especially important for down migrations where we delete rows.

		For the sake of testing, there might be a way to modifying the server
		(or queries) to perform all operations synchronously?

		Ref: https://clickhouse.com/docs/en/operations/system-tables/mutations/
		Ref: https://clickhouse.com/docs/en/sql-reference/statements/alter/#mutations
		Ref: https://clickhouse.com/blog/how-to-update-data-in-click-house/
	*/

	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)

	err = goose.UpTo(db, migrationDir, 1)
	check.NoError(t, err)
	currentVersion, err = goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 1)

	err = goose.Down(db, migrationDir)
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
	if len(results) == 0 {
		return true
	}
	check.Number(t, len(results), 1)
	check.NoError(t, rows.Close())
	check.NoError(t, rows.Err())
	return results[0].isDone == 1
}
