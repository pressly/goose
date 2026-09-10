package dialects

import (
	"fmt"

	"github.com/pressly/goose/v3/database/dialect"
)

// NewClickhouse returns a new [dialect.Querier] for Clickhouse dialect.
func NewClickhouse() dialect.Querier {
	return &clickhouse{}
}

type clickhouse struct{}

var _ dialect.Querier = (*clickhouse)(nil)

func (c *clickhouse) CreateTable(tableName string) string {
	q := `CREATE TABLE IF NOT EXISTS %s (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime default now()
	  )
	  ENGINE = MergeTree()
		ORDER BY (version_id)`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouse) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouse) DeleteVersion(tableName string) string {
	q := `ALTER TABLE %s DELETE WHERE version_id = $1 SETTINGS mutations_sync = 2`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouse) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouse) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, tableName)
}

func (c *clickhouse) GetLatestVersion(tableName string) string {
	q := `SELECT max(version_id) FROM %s`
	return fmt.Sprintf(q, tableName)
}

// TableExists returns a query that reports whether tableName exists, and raises a clear
// ClickHouse exception if it exists with an engine other than MergeTree -- i.e. it was actually
// created by the sibling clickhouse-replicated dialect, which the two must never share a table
// with.
//
// Only reachable via the Provider API ([database.NewStore]), which checks table existence before
// creating it. The legacy goose.SetDialect API (and the CLI, which uses it) never checks
// existence, so a mismatch there still goes undetected.
func (c *clickhouse) TableExists(tableName string) string {
	q := `SELECT count() > 0 AND %[2]s = 0 FROM system.tables WHERE database = currentDatabase() AND name = '%[1]s'`
	return fmt.Sprintf(q, tableName, clickhouseEngineGuardSubquery(tableName, "clickhouse", "MergeTree", "the clickhouse-replicated dialect"))
}

// clickhouseEngineGuardSubquery returns a scalar ClickHouse subquery that evaluates to 0 when
// tableName does not exist yet, or exists with the engine expectedEngine already expects it to
// have. If tableName exists with a different engine, the subquery raises a ClickHouse exception
// (via throwIf) naming the actual engine and the dialect that likely created it, instead of
// letting the caller silently reuse an incompatible table.
func clickhouseEngineGuardSubquery(tableName, thisDialect, expectedEngine, otherDialectHint string) string {
	// throwIf's message argument must be a compile-time constant in ClickHouse -- it cannot be
	// built from a runtime expression like concat(anyLast(engine), ...), so the actual engine
	// name is deliberately not included here.
	q := `(SELECT throwIf(
		count() > 0 AND anyLast(engine) != '%[3]s',
		'%[1]s: version table "%[2]s" already exists with an incompatible engine (expected %[3]s) -- it looks like it was created by %[4]s; refusing to use it to avoid corrupting migration bookkeeping'
	) FROM system.tables WHERE database = currentDatabase() AND name = '%[2]s')`
	return fmt.Sprintf(q, thisDialect, tableName, expectedEngine, otherDialectHint)
}
