package dialects

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/pressly/goose/v3/database/dialect"
)

// DefaultSparkStorageFormat is the default storage format used for the goose_db_version table in Spark SQL.
var DefaultSparkStorageFormat = "PAIMON"

// NewSpark returns a new [dialect.Querier] for the Spark SQL dialect.
// It initializes the querier using an environment-provided storage format.
func NewSpark() dialect.Querier {
	return NewSparkWithFormat(os.Getenv("GOOSE_SPARK_STORAGE_FORMAT"))
}

// NewSparkWithFormat returns a new [dialect.Querier] for the Spark SQL dialect
// using a specific storage format (e.g., "ICEBERG" or "PAIMON").
func NewSparkWithFormat(format string) dialect.Querier {
	return &spark{storageFormat: normalizeSparkStorageFormat(format)}
}

func normalizeSparkStorageFormat(format string) string {
	f := strings.ToUpper(strings.TrimSpace(format))
	switch f {
	case "PAIMON", "ICEBERG":
		return f
	default:
		return DefaultSparkStorageFormat
	}
}

// spark implements the dialect.Querier interface for Spark SQL.
type spark struct {
	storageFormat string
}

var _ dialect.Querier = (*spark)(nil)

// CreateTable generates the SQL to create the goose_db_version table.
func (s *spark) CreateTable(tableName string) string {
	// Spark SQL uses the 'USING' clause for table formats instead of 'STORED BY'.
	// Since Spark does not support AUTO_INCREMENT or IDENTITY intuitively here,
	// the 'id' field remains a standard bigint.
	var tblProps string

	switch s.storageFormat {
	case "PAIMON":
		tblProps = "\n    TBLPROPERTIES ('primary-key'='id','bucket'='1','full-compaction.delta-commits'='1','snapshot.num-retained.max'='5','snapshot.num-retained.min'='2')"
	case "ICEBERG":
		tblProps = "\n    TBLPROPERTIES ('format-version'='2')"
	}
	q := `CREATE TABLE IF NOT EXISTS %s (
		id string,
		version_id bigint,
		is_applied boolean,
		tstamp timestamp
	) USING %s%s`
	return fmt.Sprintf(q, tableName, s.storageFormat, tblProps)
}

// InsertVersion generates the SQL to insert a new migration record.
func (s *spark) InsertVersion(tableName string) string {
	id := uuid.Must(uuid.NewV7()).String()

	q := `INSERT INTO %s (id, version_id, is_applied, tstamp) VALUES ('%s', ?, ?, CURRENT_TIMESTAMP)`
	return fmt.Sprintf(q, tableName, id)
}

// DeleteVersion generates the SQL to delete a migration record.
func (s *spark) DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, tableName)
}

// GetMigrationByVersion generates the SQL to retrieve a specific migration.
func (s *spark) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

// ListMigrations generates the SQL to list all migrations.
func (s *spark) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, tableName)
}

// GetLatestVersion generates the SQL to get the maximum version_id.
func (s *spark) GetLatestVersion(tableName string) string {
	q := `SELECT max(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}
