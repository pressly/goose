package dialectquery

import "fmt"

type Ydb struct {
	Table string
}

var _ Querier = (*Ydb)(nil)

func (c *Ydb) CreateTable() string {
	return fmt.Sprintf(`CREATE TABLE %s (
		hash Uint64,
		version_id Uint64,
		is_applied UInt8,
		date Datetime,
		tstamp Datetime,

		PRIMARY KEY(hash, version_id)
	);`, c.Table)
}

func (c *Ydb) InsertVersion() string {
	q := `UPSERT INTO %s (hash, version_id, is_applied) VALUES (Digest::IntHash64(CAST($1 AS Uint64)), CAST($1 AS Uint64), $2)`
	return fmt.Sprintf(q, c.Table)
}

func (c *Ydb) DeleteVersion() string {
	q := `ALTER TABLE %s DELETE WHERE hash = Digest::IntHash64(CAST($1 AS Uint64)) AND version_id = $1 SETTINGS mutations_sync = 2`
	return fmt.Sprintf(q, c.Table)
}

func (c *Ydb) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE hash = Digest::IntHash64(CAST($1 AS Uint64)) AND version_id = $1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, c.Table)
}

func (c *Ydb) ListMigrations() string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, c.Table)
}
