package sqlite

import (
	"fmt"

	"github.com/pressly/goose/v4/internal/dialect"
)

func New(tableName string) (dialect.SQL, error) {
	return &sqlite{tableName: tableName}, nil
}

var _ dialect.SQL = (*sqlite)(nil)

type sqlite struct {
	tableName string
}

const createTable = `
CREATE TABLE %s (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	version_id INTEGER NOT NULL,
	is_applied INTEGER NOT NULL,
	tstamp TIMESTAMP DEFAULT (datetime('now'))
)
`

func (s *sqlite) CreateTable() string {
	return fmt.Sprintf(createTable, s.tableName)
}

const insertVersion = `INSERT INTO %s (version_id, is_applied) VALUES (%d, 1)`

func (s *sqlite) InsertVersion(version int64) string {
	return fmt.Sprintf(insertVersion, s.tableName, version)
}

const deleteVersion = `DELETE FROM %s WHERE version_id = %d`

func (s *sqlite) DeleteVersion(version int64) string {
	return fmt.Sprintf(deleteVersion, s.tableName, version)
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

func (s *sqlite) GetMigration(version int64) string {
	return fmt.Sprintf(getMigration, s.tableName, version)
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

func (s *sqlite) ListMigrations() string {
	return fmt.Sprintf(listMigrationsAsc, s.tableName)
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

func (s *sqlite) GetLatestMigration() string {
	return fmt.Sprintf(getLatestMigration, s.tableName)
}
