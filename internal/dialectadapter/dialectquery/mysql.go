package dialectquery

import "fmt"

type Mysql struct {
	Table string
}

var _ Querier = (*Mysql)(nil)

func (m *Mysql) CreateTable() string {
	q := `CREATE TABLE %s (
		id serial NOT NULL,
		version_id bigint NOT NULL,
		is_applied boolean NOT NULL,
		tstamp timestamp NULL default now(),
		PRIMARY KEY(id)
	)`
	return fmt.Sprintf(q, m.Table)
}

func (m *Mysql) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (?, ?)`
	return fmt.Sprintf(q, m.Table)
}

func (m *Mysql) DeleteVersion() string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, m.Table)
}

func (m *Mysql) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, m.Table)
}

func (m *Mysql) ListMigrations() string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, m.Table)
}
