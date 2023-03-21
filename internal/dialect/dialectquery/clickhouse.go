package dialectquery

import "fmt"

type Clickhouse struct {
	Table string
}

var _ Querier = (*Clickhouse)(nil)

func (c *Clickhouse) CreateTable() string {
	q := `CREATE TABLE IF NOT EXISTS %s (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime default now()
	  )
	  ENGINE = MergeTree()
		ORDER BY (date)`
	return fmt.Sprintf(q, c.Table)
}

func (c *Clickhouse) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`
	return fmt.Sprintf(q, c.Table)
}

func (c *Clickhouse) DeleteVersion() string {
	q := `ALTER TABLE %s DELETE WHERE version_id = $1 SETTINGS mutations_sync = 2`
	return fmt.Sprintf(q, c.Table)
}

func (c *Clickhouse) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, c.Table)
}

func (c *Clickhouse) ListMigrations() string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, c.Table)
}
