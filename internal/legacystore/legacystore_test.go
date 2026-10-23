package legacystore

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/pressly/goose/v3/database/dialect"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

type testTableCreatorQuerier struct {
	mockQuerier
	createTableContextCalled bool
	receivedCtx              context.Context
	receivedDB               dialect.DBTxConn
	receivedTableName        string
}

func (c *testTableCreatorQuerier) CreateTableContext(ctx context.Context, db dialect.DBTxConn, tableName string) error {
	c.createTableContextCalled = true
	c.receivedCtx = ctx
	c.receivedDB = db
	c.receivedTableName = tableName
	q := fmt.Sprintf(`CREATE TABLE %s (
		id INTEGER PRIMARY KEY,
		version_id INTEGER NOT NULL,
		is_applied BOOLEAN NOT NULL,
		tstamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`, tableName)
	_, err := db.ExecContext(ctx, q)
	return err
}

type mockQuerier struct{}

func (m *mockQuerier) CreateTable(tableName string) string {
	return fmt.Sprintf(`CREATE TABLE %s (
		id INTEGER PRIMARY KEY,
		version_id INTEGER NOT NULL,
		is_applied BOOLEAN NOT NULL,
		tstamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`, tableName)
}
func (m *mockQuerier) InsertVersion(tableName string) string {
	return fmt.Sprintf("INSERT INTO %s (version_id, is_applied) VALUES (?, ?);", tableName)
}
func (m *mockQuerier) DeleteVersion(tableName string) string {
	return fmt.Sprintf("DELETE FROM %s WHERE version_id = ?;", tableName)
}
func (m *mockQuerier) GetMigrationByVersion(tableName string) string {
	return fmt.Sprintf("SELECT tstamp, is_applied FROM %s WHERE version_id = ? ORDER BY tstamp DESC LIMIT 1;", tableName)
}
func (m *mockQuerier) ListMigrations(tableName string) string {
	return fmt.Sprintf("SELECT version_id, is_applied FROM %s ORDER BY id DESC;", tableName)
}
func (m *mockQuerier) GetLatestVersion(tableName string) string {
	return fmt.Sprintf("SELECT max(version_id) FROM %s;", tableName)
}

func TestLegacyStoreTableCreator(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	querier := &testTableCreatorQuerier{}
	s := &store{querier: querier}

	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()

	tableName := "legacy_custom_table"
	err = s.CreateVersionTable(ctx, tx, tableName)
	require.NoError(t, err)

	require.True(t, querier.createTableContextCalled)
	require.Equal(t, ctx, querier.receivedCtx)
	require.Equal(t, tx, querier.receivedDB)
	require.Equal(t, tableName, querier.receivedTableName)
}
