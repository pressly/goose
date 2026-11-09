package dialects

import (
	"fmt"

	"github.com/pressly/goose/v3/database/dialect"
)

// NewSpanner returns a [dialect.QuerierExtender] for Spanner dialect.
func NewSpanner() dialect.QuerierExtender {
	return &spanner{}
}

type spanner struct{}

var _ dialect.QuerierExtender = (*spanner)(nil)

func (s *spanner) CreateTable(tableName string) string {
	q := `CREATE TABLE %s (
		version_id INT64 NOT NULL,
		is_applied BOOL NOT NULL,
		tstamp TIMESTAMP DEFAULT (CURRENT_TIMESTAMP()),
	) PRIMARY KEY(version_id)`
	return fmt.Sprintf(q, tableName)
}

func (s *spanner) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (?, ?)`
	return fmt.Sprintf(q, tableName)
}

func (s *spanner) DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, tableName)
}

func (s *spanner) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (s *spanner) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied from %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, tableName)
}

func (s *spanner) GetLatestVersion(tableName string) string {
	q := `SELECT MAX(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}

func (s *spanner) TableExists(tableName string) string {
	// Spanner's default schema is an empty string, so parseTableIdentifier
	// returns ("", tableName) for unqualified names, which is correct.
	schemaName, tableName := parseTableIdentifier(tableName)
	q := `SELECT EXISTS ( SELECT 1 FROM information_schema.tables WHERE table_schema = '%s' AND table_name = '%s' )`
	return fmt.Sprintf(q, schemaName, tableName)
}
