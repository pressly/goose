package dialectquery

import "fmt"

type tidb struct {
	table string
}

func (t *tidb) CreateTable() string {
	q := `CREATE TABLE %s (
		id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT UNIQUE,
		version_id bigint NOT NULL,
		is_applied boolean NOT NULL,
		tstamp timestamp NULL default now(),
		PRIMARY KEY(id)
	)`
	return fmt.Sprintf(q, t.table)
}

func (t *tidb) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (?, ?)`
	return fmt.Sprintf(q, t.table)
}

func (t *tidb) DeleteVersion() string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, t.table)
}

func (t *tidb) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, t.table)
}

func (t *tidb) ListMigrations() string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, t.table)
}
