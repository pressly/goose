package dialects

import (
	"fmt"

	"github.com/pressly/goose/v3/database/dialect"
)

// NewDuckDB returns a [dialect.Querier] for DuckDB dialect.
func NewDuckDB() dialect.Querier {
	return &duckdb{}
}

type duckdb struct{}

var _ dialect.Querier = (*duckdb)(nil)

func (d *duckdb) CreateTable(tableName string) string {
	q := `CREATE SEQUENCE IF NOT EXISTS %s_id_seq START 1;
	CREATE TABLE %s (
		id INTEGER PRIMARY KEY DEFAULT nextval('%s_id_seq'),
		version_id BIGINT NOT NULL,
		is_applied BOOLEAN NOT NULL,
		tstamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`
	return fmt.Sprintf(q, tableName, tableName, tableName)
}

func (d *duckdb) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (?, ?)`
	return fmt.Sprintf(q, tableName)
}

func (d *duckdb) DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, tableName)
}

func (d *duckdb) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (d *duckdb) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, tableName)
}

func (d *duckdb) GetLatestVersion(tableName string) string {
	q := `SELECT MAX(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}
