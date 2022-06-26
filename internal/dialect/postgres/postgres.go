package postgres

import (
	"fmt"

	"github.com/pressly/goose/v3/internal/dialect"
)

func New(tableName string) (dialect.SQL, error) {
	return &postgres{tableName: tableName}, nil
}

var _ dialect.SQL = (*postgres)(nil)

type postgres struct {
	tableName string
}

const createTable = `
CREATE TABLE %s (
	id serial NOT NULL,
	version_id bigint NOT NULL,
	is_applied boolean NOT NULL,
	tstamp timestamp NULL default now(),
	PRIMARY KEY(id)
  )
`

func (p *postgres) CreateTable() string {
	return fmt.Sprintf(createTable, p.tableName)
}

const insertVersion = `INSERT INTO %s (version_id, is_applied) VALUES (%d, true)`

func (p *postgres) InsertVersion(version int64) string {
	return fmt.Sprintf(insertVersion, p.tableName, version)
}

const deleteVersion = `DELETE FROM %s WHERE version_id = %d`

func (p *postgres) DeleteVersion(version int64) string {
	return fmt.Sprintf(deleteVersion, p.tableName, version)
}

const getMigration = `
SELECT
	id,
	version_id,
	tstamp
FROM
	%s
WHERE
	version_id = %d
`

func (p *postgres) GetMigration(version int64) string {
	return fmt.Sprintf(getMigration, p.tableName, version)
}

const listMigrationsAsc = `
SELECT
	id,
	version_id,
	tstamp
FROM
	%s
ORDER BY
	id ASC
`

func (p *postgres) ListMigrations() string {
	return fmt.Sprintf(listMigrationsAsc, p.tableName)
}

const getLatestMigration = `
SELECT
	id,
	version_id,
	tstamp
FROM
	%s
ORDER BY
	id DESC
LIMIT 1
`

func (p *postgres) GetLatestMigration() string {
	return fmt.Sprintf(getLatestMigration, p.tableName)
}
