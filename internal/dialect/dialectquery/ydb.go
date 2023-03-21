package dialectquery

import "fmt"

type Ydb struct {
	Table string
}

var _ Querier = (*Ydb)(nil)

func (c *Ydb) CreateTable() string {
	q := `CREATE TABLE %s (
		id Uint64,
		version_id Uint64,
		is_applied Bool,
		tstamp Datetime,
		PRIMARY KEY(id, version_id)
	)`
	return fmt.Sprintf(q, c.Table)
}

func (c *Ydb) InsertVersion() string {
	q := `
		INSERT INTO versions (id, version_id, is_applied, tstamp) 
		VALUES (Digest::NumericHash($version_id), $version_id, $is_applied, CurrentUtcDatetime());`
	return fmt.Sprintf(q, c.Table, c.Table)
}

func (c *Ydb) DeleteVersion() string {
	q := `ALTER TABLE %s DELETE WHERE version_id = $1`
	return fmt.Sprintf(q, c.Table)
}

func (c *Ydb) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, c.Table)
}

func (c *Ydb) ListMigrations() string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, c.Table)
}
