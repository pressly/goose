package dialects

import (
	"fmt"

	"github.com/pressly/goose/v3/database/dialect"
)

// NewTDengine returns a [dialect.Querier] for TDengine dialect.
func NewTDengine() dialect.Querier {
	return &tdengine{}
}

type tdengine struct{}

var _ dialect.Querier = (*tdengine)(nil)

func (t *tdengine) SupportsTx() bool {
	return false
}

func (t *tdengine) CreateTable(tableName string) string {
	q := `CREATE TABLE IF NOT EXISTS %s (
		version_id TIMESTAMP,
		is_applied BOOL,
		tstamp TIMESTAMP
	)`
	return fmt.Sprintf(q, tableName)
}

func (t *tdengine) InsertVersion(tableName string) string {
	q := `INSERT INTO %s VALUES (?, ?, now())`
	return fmt.Sprintf(q, tableName)
}

func (t *tdengine) DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, tableName)
}

func (t *tdengine) GetMigrationByVersion(tableName string) string {
	q := `SELECT version_id, is_applied FROM %s WHERE version_id=? ORDER BY version_id DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (t *tdengine) ListMigrations(tableName string) string {
	q := `SELECT CAST(version_id AS BIGINT), is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, tableName)
}

func (t *tdengine) GetLatestVersion(tableName string) string {
	q := `SELECT CAST(MAX(version_id) AS BIGINT) FROM %s`
	return fmt.Sprintf(q, tableName)
}
