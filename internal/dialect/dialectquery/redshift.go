package dialectquery

import "fmt"

type Redshift struct {
	Table string
}

var _ Querier = (*Redshift)(nil)

func (r *Redshift) CreateTable() string {
	q := `CREATE TABLE %s (
		id integer NOT NULL identity(1, 1),
		version_id bigint NOT NULL,
		is_applied boolean NOT NULL,
		tstamp timestamp NULL default sysdate,
		PRIMARY KEY(id)
	)`
	return fmt.Sprintf(q, r.Table)
}

func (r *Redshift) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`
	return fmt.Sprintf(q, r.Table)
}

func (r *Redshift) DeleteVersion() string {
	q := `DELETE FROM %s WHERE version_id=$1`
	return fmt.Sprintf(q, r.Table)
}

func (r *Redshift) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, r.Table)
}

func (r *Redshift) ListMigrations() string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, r.Table)
}
