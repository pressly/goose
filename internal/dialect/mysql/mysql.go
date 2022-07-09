package mysql

import (
	"fmt"

	"github.com/pressly/goose/v3/internal/dialect"
)

func New(tableName string) (dialect.SQL, error) {
	return &mysql{tableName: tableName}, nil
}

var _ dialect.SQL = (*mysql)(nil)

type mysql struct {
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

func (m *mysql) CreateTable() string {
	return fmt.Sprintf(createTable, m.tableName)
}

const insertVersion = `INSERT INTO %s (version_id, is_applied) VALUES (%d, true)`

func (m *mysql) InsertVersion(version int64) string {
	return fmt.Sprintf(insertVersion, m.tableName, version)
}

const deleteVersion = `DELETE FROM %s WHERE version_id = %d`

func (m *mysql) DeleteVersion(version int64) string {
	return fmt.Sprintf(deleteVersion, m.tableName, version)
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

func (m *mysql) GetMigration(version int64) string {
	return fmt.Sprintf(getMigration, m.tableName, version)
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

func (m *mysql) ListMigrations() string {
	return fmt.Sprintf(listMigrationsAsc, m.tableName)
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

func (m *mysql) GetLatestMigration() string {
	return fmt.Sprintf(getLatestMigration, m.tableName)
}
