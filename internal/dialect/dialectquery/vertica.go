package dialectquery

import "fmt"

type Vertica struct {
	Table string
}

var _ Querier = (*Vertica)(nil)

func (v *Vertica) CreateTable() string {
	q := `CREATE TABLE %s (
		id identity(1,1) NOT NULL,
		version_id bigint NOT NULL,
		is_applied boolean NOT NULL,
		tstamp timestamp NULL default now(),
		PRIMARY KEY(id)
	)`
	return fmt.Sprintf(q, v.Table)
}

func (v *Vertica) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (?, ?)`
	return fmt.Sprintf(q, v.Table)
}

func (v *Vertica) DeleteVersion() string {
	q := `DELETE FROM %s WHERE version_id=?`
	return fmt.Sprintf(q, v.Table)
}

func (v *Vertica) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=? ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, v.Table)
}

func (v *Vertica) ListMigrations() string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, v.Table)
}
