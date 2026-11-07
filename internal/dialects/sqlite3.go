package dialects

import (
	"fmt"

	"github.com/pressly/goose/v3/database/dialect"
)

// NewSqlite3 returns a [dialect.QuerierExtender] for SQLite3 dialect.
func NewSqlite3() dialect.QuerierExtender {
	return &sqlite3{}
}

type sqlite3 struct{}

var _ dialect.QuerierExtender = (*sqlite3)(nil)

func (s *sqlite3) CreateTable(tableName string) string {
	q := `CREATE TABLE %s (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		version_id INTEGER NOT NULL,
		is_applied INTEGER NOT NULL,
		tstamp TIMESTAMP DEFAULT (datetime('now'))
	)`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlite3) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (?, ?)`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlite3) DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlite3) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlite3) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlite3) GetLatestVersion(tableName string) string {
	q := `SELECT MAX(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlite3) TableExists(tableName string) string {
	schemaName, tableName := parseTableIdentifier(tableName)
	if schemaName != "" {
		q := `SELECT EXISTS ( SELECT 1 FROM %s.sqlite_master WHERE type = 'table' AND name = '%s' )`
		return fmt.Sprintf(q, schemaName, tableName)
	}
	q := `SELECT EXISTS ( SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = '%s' )`
	return fmt.Sprintf(q, tableName)
}
