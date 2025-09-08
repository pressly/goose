package lock

import (
	"fmt"
	"testing"

	"github.com/pressly/goose/v3/database"
	"github.com/stretchr/testify/require"
)

func TestNewLockStoreForDialect(t *testing.T) {
	t.Run("supported_dialects", func(t *testing.T) {
		tests := []struct {
			name     string
			dialect  database.Dialect
			expected string
		}{
			{
				name:     "postgres",
				dialect:  database.DialectPostgres,
				expected: "*lock.PostgresLockQuerier",
			},
			{
				name:     "sqlite",
				dialect:  database.DialectSQLite3,
				expected: "*lock.SQLiteLockQuerier",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				store, err := NewLockStoreForDialect(tt.dialect, "test_table")
				require.NoError(t, err)
				require.NotNil(t, store)
				
				lockStore := store.(*lockStore)
				require.Equal(t, "test_table", lockStore.tableName)
				
				querierType := fmt.Sprintf("%T", lockStore.querier)
				require.Equal(t, tt.expected, querierType)
			})
		}
	})

	t.Run("unsupported_dialects", func(t *testing.T) {
		unsupportedDialects := []struct {
			name    string
			dialect database.Dialect
		}{
			{"mysql", database.DialectMySQL},
			{"mssql", database.DialectMSSQL},
			{"clickhouse", database.DialectClickHouse},
			{"aurora_dsql", database.DialectAuroraDSQL},
			{"redshift", database.DialectRedshift},
			{"starrocks", database.DialectStarrocks},
			{"tidb", database.DialectTiDB},
			{"turso", database.DialectTurso},
			{"ydb", database.DialectYdB},
			{"vertica", database.DialectVertica},
			{"custom", database.DialectCustom},
			{"unknown", "unknown"},
		}

		for _, tt := range unsupportedDialects {
			t.Run(tt.name, func(t *testing.T) {
				store, err := NewLockStoreForDialect(tt.dialect, "test_table")
				require.Error(t, err)
				require.Nil(t, store)
				require.Contains(t, err.Error(), "table-based locking not implemented for dialect")
				require.Contains(t, err.Error(), string(tt.dialect))
			})
		}
	})

	t.Run("empty_table_name_uses_default", func(t *testing.T) {
		store, err := NewLockStoreForDialect(database.DialectPostgres, "")
		require.NoError(t, err)
		require.NotNil(t, store)
		
		lockStore := store.(*lockStore)
		require.Equal(t, DefaultLockTableName, lockStore.tableName)
	})

	t.Run("custom_table_name", func(t *testing.T) {
		customTableName := "my_custom_lock_table"
		store, err := NewLockStoreForDialect(database.DialectSQLite3, customTableName)
		require.NoError(t, err)
		require.NotNil(t, store)
		
		lockStore := store.(*lockStore)
		require.Equal(t, customTableName, lockStore.tableName)
	})
}

func TestDialectSpecificSQL(t *testing.T) {
	t.Run("postgres_sql_generation", func(t *testing.T) {
		querier := NewPostgresLockQuerier()
		tableName := "test_lock_table"
		
		// Test CreateLockTable
		createSQL := querier.CreateLockTable(tableName)
		require.Contains(t, createSQL, "CREATE TABLE IF NOT EXISTS "+tableName)
		require.Contains(t, createSQL, "id INTEGER PRIMARY KEY")
		require.Contains(t, createSQL, "locked INTEGER NOT NULL DEFAULT 0")
		require.Contains(t, createSQL, "lock_granted TIMESTAMP")
		require.Contains(t, createSQL, "last_heartbeat TIMESTAMP")
		require.Contains(t, createSQL, "locked_by TEXT")
		
		// Test AcquireLock
		acquireSQL := querier.AcquireLock(tableName)
		require.Contains(t, acquireSQL, "UPDATE "+tableName)
		require.Contains(t, acquireSQL, "$1")
		require.Contains(t, acquireSQL, "NOW()")
		require.Contains(t, acquireSQL, "WHERE id = 1 AND locked = 0")
		
		// Test InsertInitialLock
		insertSQL := querier.InsertInitialLock(tableName)
		require.Contains(t, insertSQL, "INSERT INTO "+tableName)
		require.Contains(t, insertSQL, "ON CONFLICT (id) DO NOTHING")
		require.Contains(t, insertSQL, "$1")
		require.Contains(t, insertSQL, "NOW()")
		
		// Test ReleaseLock
		releaseSQL := querier.ReleaseLock(tableName)
		require.Contains(t, releaseSQL, "UPDATE "+tableName)
		require.Contains(t, releaseSQL, "SET locked = 0")
		require.Contains(t, releaseSQL, "WHERE id = 1")
		
		// Test UpdateHeartbeat
		heartbeatSQL := querier.UpdateHeartbeat(tableName)
		require.Contains(t, heartbeatSQL, "UPDATE "+tableName)
		require.Contains(t, heartbeatSQL, "SET last_heartbeat = NOW()")
		require.Contains(t, heartbeatSQL, "WHERE id = 1 AND locked = 1")
		
		// Test CleanupStaleLocks
		cleanupSQL := querier.CleanupStaleLocks(tableName)
		require.Contains(t, cleanupSQL, "UPDATE "+tableName)
		require.Contains(t, cleanupSQL, "SET locked = 0")
		require.Contains(t, cleanupSQL, "WHERE id = 1 AND locked = 1")
		require.Contains(t, cleanupSQL, "last_heartbeat < NOW() - INTERVAL '$1 seconds'")
	})
	
	t.Run("sqlite_sql_generation", func(t *testing.T) {
		querier := NewSQLiteLockQuerier()
		tableName := "test_lock_table"
		
		// Test CreateLockTable
		createSQL := querier.CreateLockTable(tableName)
		require.Contains(t, createSQL, "CREATE TABLE IF NOT EXISTS "+tableName)
		require.Contains(t, createSQL, "id INTEGER PRIMARY KEY")
		require.Contains(t, createSQL, "locked INTEGER NOT NULL DEFAULT 0")
		require.Contains(t, createSQL, "lock_granted TEXT")
		require.Contains(t, createSQL, "last_heartbeat TEXT")
		require.Contains(t, createSQL, "locked_by TEXT")
		
		// Test AcquireLock
		acquireSQL := querier.AcquireLock(tableName)
		require.Contains(t, acquireSQL, "UPDATE "+tableName)
		require.Contains(t, acquireSQL, "?")
		require.Contains(t, acquireSQL, "datetime('now')")
		require.Contains(t, acquireSQL, "WHERE id = 1 AND locked = 0")
		
		// Test InsertInitialLock
		insertSQL := querier.InsertInitialLock(tableName)
		require.Contains(t, insertSQL, "INSERT OR IGNORE INTO "+tableName)
		require.Contains(t, insertSQL, "?")
		require.Contains(t, insertSQL, "datetime('now')")
		
		// Test ReleaseLock
		releaseSQL := querier.ReleaseLock(tableName)
		require.Contains(t, releaseSQL, "UPDATE "+tableName)
		require.Contains(t, releaseSQL, "SET locked = 0")
		require.Contains(t, releaseSQL, "WHERE id = 1")
		
		// Test UpdateHeartbeat
		heartbeatSQL := querier.UpdateHeartbeat(tableName)
		require.Contains(t, heartbeatSQL, "UPDATE "+tableName)
		require.Contains(t, heartbeatSQL, "SET last_heartbeat = datetime('now')")
		require.Contains(t, heartbeatSQL, "WHERE id = 1 AND locked = 1")
		
		// Test CleanupStaleLocks
		cleanupSQL := querier.CleanupStaleLocks(tableName)
		require.Contains(t, cleanupSQL, "UPDATE "+tableName)
		require.Contains(t, cleanupSQL, "SET locked = 0")
		require.Contains(t, cleanupSQL, "WHERE id = 1 AND locked = 1")
		require.Contains(t, cleanupSQL, "last_heartbeat < datetime('now', '-' || ? || ' seconds')")
	})
}

func TestTableNameParameterization(t *testing.T) {
	testCases := []struct {
		tableName string
		desc      string
	}{
		{"simple_table", "simple table name"},
		{"table_with_underscores", "table with underscores"},
		{"TableWithCamelCase", "table with camel case"},
		{"table123", "table with numbers"},
		{"a_very_long_table_name_that_should_still_work_properly", "very long table name"},
	}

	t.Run("postgres_table_names", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				querier := NewPostgresLockQuerier()
				
				// Verify table name appears in all SQL statements
				require.Contains(t, querier.CreateLockTable(tc.tableName), tc.tableName)
				require.Contains(t, querier.AcquireLock(tc.tableName), tc.tableName)
				require.Contains(t, querier.InsertInitialLock(tc.tableName), tc.tableName)
				require.Contains(t, querier.ReleaseLock(tc.tableName), tc.tableName)
				require.Contains(t, querier.UpdateHeartbeat(tc.tableName), tc.tableName)
				require.Contains(t, querier.CleanupStaleLocks(tc.tableName), tc.tableName)
			})
		}
	})

	t.Run("sqlite_table_names", func(t *testing.T) {
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				querier := NewSQLiteLockQuerier()
				
				// Verify table name appears in all SQL statements
				require.Contains(t, querier.CreateLockTable(tc.tableName), tc.tableName)
				require.Contains(t, querier.AcquireLock(tc.tableName), tc.tableName)
				require.Contains(t, querier.InsertInitialLock(tc.tableName), tc.tableName)
				require.Contains(t, querier.ReleaseLock(tc.tableName), tc.tableName)
				require.Contains(t, querier.UpdateHeartbeat(tc.tableName), tc.tableName)
				require.Contains(t, querier.CleanupStaleLocks(tc.tableName), tc.tableName)
			})
		}
	})
}

func TestDialectConstants(t *testing.T) {
	t.Run("default_lock_table_name_is_defined", func(t *testing.T) {
		require.Equal(t, "goose_db_lock", DefaultLockTableName)
		require.NotEmpty(t, DefaultLockTableName)
	})
}

func TestDialectQuerierInterface(t *testing.T) {
	t.Run("postgres_querier_implements_interface", func(t *testing.T) {
		var querier LockQuerier = NewPostgresLockQuerier()
		require.NotNil(t, querier)
		
		// Test all interface methods return non-empty strings
		tableName := "test"
		require.NotEmpty(t, querier.CreateLockTable(tableName))
		require.NotEmpty(t, querier.AcquireLock(tableName))
		require.NotEmpty(t, querier.InsertInitialLock(tableName))
		require.NotEmpty(t, querier.ReleaseLock(tableName))
		require.NotEmpty(t, querier.UpdateHeartbeat(tableName))
		require.NotEmpty(t, querier.CleanupStaleLocks(tableName))
	})

	t.Run("sqlite_querier_implements_interface", func(t *testing.T) {
		var querier LockQuerier = NewSQLiteLockQuerier()
		require.NotNil(t, querier)
		
		// Test all interface methods return non-empty strings
		tableName := "test"
		require.NotEmpty(t, querier.CreateLockTable(tableName))
		require.NotEmpty(t, querier.AcquireLock(tableName))
		require.NotEmpty(t, querier.InsertInitialLock(tableName))
		require.NotEmpty(t, querier.ReleaseLock(tableName))
		require.NotEmpty(t, querier.UpdateHeartbeat(tableName))
		require.NotEmpty(t, querier.CleanupStaleLocks(tableName))
	})
}