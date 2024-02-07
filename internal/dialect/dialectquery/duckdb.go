package dialectquery

import "fmt"

type Duckdb struct{}

var _ Querier = (*Duckdb)(nil)

func (s *Duckdb) CreateTable(tableName string) string {
	q := `
	CREATE SEQUENCE %s_id;
	CREATE TABLE %s (
		id BIGINT PRIMARY KEY DEFAULT NEXTVAL('%s_id'),
		version_id BIGINT NOT NULL,
		is_applied BOOLEAN NOT NULL,
		tstamp TIMESTAMP DEFAULT (current_date())
	)`
	return fmt.Sprintf(q, tableName, tableName, tableName)
}

func (s *Duckdb) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (?, ?)`
	return fmt.Sprintf(q, tableName)
}

func (s *Duckdb) DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, tableName)
}

func (s *Duckdb) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (s *Duckdb) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, tableName)
}
