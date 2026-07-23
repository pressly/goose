package dialects

import (
	"strings"
	"testing"
)

func TestDuckDBCreateTable(t *testing.T) {
	d := NewDuckDB()
	tableName := "goose_db_version"

	sql := d.CreateTable(tableName)

	// Verify it creates a sequence first
	if !strings.Contains(sql, "CREATE SEQUENCE IF NOT EXISTS goose_db_version_id_seq START 1") {
		t.Errorf("CreateTable should create a sequence, got: %s", sql)
	}

	// Verify it creates a table with nextval for the id column
	if !strings.Contains(sql, "CREATE TABLE goose_db_version") {
		t.Errorf("CreateTable should create the table, got: %s", sql)
	}

	// Verify it uses nextval for the id default
	if !strings.Contains(sql, "DEFAULT nextval('goose_db_version_id_seq')") {
		t.Errorf("CreateTable should use nextval for id default, got: %s", sql)
	}

	// Verify required columns exist
	requiredColumns := []string{
		"id INTEGER PRIMARY KEY",
		"version_id BIGINT NOT NULL",
		"is_applied BOOLEAN NOT NULL",
		"tstamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
	}
	for _, col := range requiredColumns {
		if !strings.Contains(sql, col) {
			t.Errorf("CreateTable should contain %q, got: %s", col, sql)
		}
	}
}

func TestDuckDBInsertVersion(t *testing.T) {
	d := NewDuckDB()
	tableName := "goose_db_version"

	sql := d.InsertVersion(tableName)

	// Should use $1, $2 placeholders (PostgreSQL style)
	expected := "INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, $2)"
	if sql != expected {
		t.Errorf("InsertVersion = %q, want %q", sql, expected)
	}
}

func TestDuckDBDeleteVersion(t *testing.T) {
	d := NewDuckDB()
	tableName := "goose_db_version"

	sql := d.DeleteVersion(tableName)

	expected := "DELETE FROM goose_db_version WHERE version_id=$1"
	if sql != expected {
		t.Errorf("DeleteVersion = %q, want %q", sql, expected)
	}
}

func TestDuckDBGetMigrationByVersion(t *testing.T) {
	d := NewDuckDB()
	tableName := "goose_db_version"

	sql := d.GetMigrationByVersion(tableName)

	expected := "SELECT tstamp, is_applied FROM goose_db_version WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1"
	if sql != expected {
		t.Errorf("GetMigrationByVersion = %q, want %q", sql, expected)
	}
}

func TestDuckDBListMigrations(t *testing.T) {
	d := NewDuckDB()
	tableName := "goose_db_version"

	sql := d.ListMigrations(tableName)

	expected := "SELECT version_id, is_applied FROM goose_db_version ORDER BY id DESC"
	if sql != expected {
		t.Errorf("ListMigrations = %q, want %q", sql, expected)
	}
}

func TestDuckDBGetLatestVersion(t *testing.T) {
	d := NewDuckDB()
	tableName := "goose_db_version"

	sql := d.GetLatestVersion(tableName)

	expected := "SELECT MAX(version_id) FROM goose_db_version"
	if sql != expected {
		t.Errorf("GetLatestVersion = %q, want %q", sql, expected)
	}
}

func TestDuckDBSequenceNaming(t *testing.T) {
	d := NewDuckDB()

	// Test with custom table name to ensure sequence name is derived correctly
	testCases := []struct {
		tableName    string
		expectedSeq  string
	}{
		{"goose_db_version", "goose_db_version_id_seq"},
		{"custom_migrations", "custom_migrations_id_seq"},
		{"my_schema.goose_db_version", "my_schema.goose_db_version_id_seq"},
	}

	for _, tc := range testCases {
		sql := d.CreateTable(tc.tableName)
		if !strings.Contains(sql, tc.expectedSeq) {
			t.Errorf("CreateTable(%q) should use sequence %q, got: %s", tc.tableName, tc.expectedSeq, sql)
		}
	}
}
