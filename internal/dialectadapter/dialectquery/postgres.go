package dialectquery

import "fmt"

type Postgres struct {
	Table string
}

var _ Querier = (*Postgres)(nil)

func (p *Postgres) CreateTable() string {
	q := `CREATE TABLE %s (
		id serial NOT NULL,
		version_id bigint NOT NULL,
		is_applied boolean NOT NULL,
		tstamp timestamp NULL default now(),
		PRIMARY KEY(id)
	)`
	return fmt.Sprintf(q, p.Table)
}

func (p *Postgres) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`
	return fmt.Sprintf(q, p.Table)
}

func (p *Postgres) DeleteVersion() string {
	q := `DELETE FROM %s WHERE version_id=$1`
	return fmt.Sprintf(q, p.Table)
}

func (p *Postgres) GetMigrationByVersion() string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id=$1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, p.Table)
}

func (p *Postgres) ListMigrations() string {
	q := `SELECT version_id, is_applied from %s ORDER BY id DESC`
	return fmt.Sprintf(q, p.Table)
}

// AdvisoryLockSession returns the query to lock the database using an exclusive session level
// advisory lock.
func (p *Postgres) AdvisoryLockSession(id int64) string {
	q := `SELECT pg_advisory_lock(%d)`
	return fmt.Sprintf(q, id)
}

func (p *Postgres) TryAdvisoryLockSession(id int64) string {
	q := `SELECT pg_try_advisory_lock(%d)`
	return fmt.Sprintf(q, id)
}

// AdvisoryUnlockSession returns the query to release an exclusive session level advisory lock.
func (p *Postgres) AdvisoryUnlockSession(id int64) string {
	q := `SELECT pg_advisory_unlock(%d)`
	return fmt.Sprintf(q, id)
}

// AdvisoryLockTransaction returns the query to lock the database using an exclusive transaction
// level advisory lock.
//
// The lock is automatically released at the end of the current transaction and cannot be released
// explicitly.
func (p *Postgres) AdvisoryLockTransaction(id int64) string {
	q := `SELECT pg_advisory_xact_lock(%d)`
	return fmt.Sprintf(q, id)
}
