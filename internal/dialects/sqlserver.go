package dialects

import (
	"fmt"

	"github.com/pressly/goose/v3/database/dialect"
)

// NewSqlserver returns a [dialect.QuerierExtender] for SQL Server dialect.
func NewSqlserver() dialect.QuerierExtender {
	return &sqlserver{}
}

type sqlserver struct{}

var _ dialect.QuerierExtender = (*sqlserver)(nil)

func (s *sqlserver) CreateTable(tableName string) string {
	q := `CREATE TABLE %s (
		id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
		version_id BIGINT NOT NULL,
		is_applied BIT NOT NULL,
		tstamp DATETIME NULL DEFAULT CURRENT_TIMESTAMP
	)`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlserver) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (@p1, @p2)`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlserver) DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=@p1`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlserver) GetMigrationByVersion(tableName string) string {
	q := `SELECT TOP 1 tstamp, is_applied FROM %s WHERE version_id=@p1 ORDER BY tstamp DESC`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlserver) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY id DESC`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlserver) GetLatestVersion(tableName string) string {
	q := `SELECT MAX(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}

func (s *sqlserver) TableExists(tableName string) string {
	schemaName, tableName := parseTableIdentifier(tableName)
	if schemaName != "" {
		q := `SELECT CASE WHEN EXISTS ( SELECT 1 FROM information_schema.tables WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s' ) THEN 1 ELSE 0 END`
		return fmt.Sprintf(q, schemaName, tableName)
	}
	q := `SELECT CASE WHEN EXISTS ( SELECT 1 FROM information_schema.tables WHERE (SCHEMA_NAME() IS NULL OR TABLE_SCHEMA = SCHEMA_NAME()) AND TABLE_NAME = '%s' ) THEN 1 ELSE 0 END`
	return fmt.Sprintf(q, tableName)
}
