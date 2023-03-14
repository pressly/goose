package dialectquery

import "fmt"

type postgres struct {
	table string
}

func (p *postgres) CreateTable() string {
	q := `CREATE TABLE %s (
		id serial NOT NULL,
		version_id bigint NOT NULL,
		is_applied boolean NOT NULL,
		tstamp timestamp NULL default now(),
		PRIMARY KEY(id)
	)`
	return fmt.Sprintf(q, p.table)
}

func (p *postgres) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`
	return fmt.Sprintf(q, p.table)
}

func (p *postgres) DeleteVersion() string {
	q := `DELETE FROM %s WHERE version_id=$1`
	return fmt.Sprintf(q, p.table)
}

func (p *postgres) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, p.table)
}

func (p *postgres) ListMigrations() string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, p.table)
}
