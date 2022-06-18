package clickhouse_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/testdb"
)

func TestClickHouse(t *testing.T) {
	t.Parallel()

	migrationDir := filepath.Join("testdata", "migrations")
	db, cleanup, err := testdb.NewClickHouse()
	check.NoError(t, err)
	t.Cleanup(cleanup)

	goose.SetDialect("clickhouse")

	currentVersion, err := goose.GetDBVersion(db)
	check.NoError(t, err)
	check.Number(t, currentVersion, 0)

	err = goose.Up(db, migrationDir)
	check.NoError(t, err)

	q := fmt.Sprintf(`SELECT version_id, is_applied FROM %s ORDER BY tstamp DESC LIMIT 1`, goose.TableName())
	var versionID int64
	var isApplied bool
	err = db.QueryRow(q).Scan(&versionID, &isApplied)
	check.NoError(t, err)
	check.Number(t, versionID, 1)
	check.Bool(t, isApplied, true)

	// TODO(mf): figure out how down migrations are handled in ClickHouse.
	// SETTINGS mutations_sync = 0 is the default, which means the operation
	// is done async. We care, because we want to test the down migration
	// and confirm the table and migration history got removed.
	//
	// One option is to loop N times / seconds, checking to see if the
	// operation has been completed. But there must be a better way.
}
