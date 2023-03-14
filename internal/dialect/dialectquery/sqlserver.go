package dialectquery

import "fmt"

type sqlserver struct {
	table string
}

func (s *sqlserver) CreateTable() string {
	q := `CREATE TABLE %s (
		id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
		version_id BIGINT NOT NULL,
		is_applied BIT NOT NULL,
		tstamp DATETIME NULL DEFAULT CURRENT_TIMESTAMP
	)`
	return fmt.Sprintf(q, s.table)
}

func (s *sqlserver) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (@p1, @p2)`
	return fmt.Sprintf(q, s.table)
}

func (s *sqlserver) DeleteVersion() string {
	q := `DELETE FROM %s WHERE version_id=@p1`
	return fmt.Sprintf(q, s.table)
}

func (s *sqlserver) GetMigrationByVersion() string {
	q := `
WITH Migrations AS
(
	SELECT tstamp, is_applied,
	ROW_NUMBER() OVER (ORDER BY tstamp) AS 'RowNumber'
	FROM %s
	WHERE version_id=@p1
)
SELECT tstamp, is_applied
FROM Migrations
WHERE RowNumber BETWEEN 1 AND 2
ORDER BY tstamp DESC
`
	return fmt.Sprintf(q, s.table)
}

func (s *sqlserver) ListMigrations() string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY id DESC`
	return fmt.Sprintf(q, s.table)
}
