package dialectquery

import "fmt"

type Sqlserver struct {
	Table string
}

var _ Querier = (*Sqlserver)(nil)

func (s *Sqlserver) CreateTable() string {
	q := `CREATE TABLE %s (
		id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
		version_id BIGINT NOT NULL,
		is_applied BIT NOT NULL,
		tstamp DATETIME NULL DEFAULT CURRENT_TIMESTAMP
	)`
	return fmt.Sprintf(q, s.Table)
}

func (s *Sqlserver) InsertVersion() string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (@p1, @p2)`
	return fmt.Sprintf(q, s.Table)
}

func (s *Sqlserver) DeleteVersion() string {
	q := `DELETE FROM %s WHERE version_id=@p1`
	return fmt.Sprintf(q, s.Table)
}

func (s *Sqlserver) GetMigrationByVersion() string {
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
	return fmt.Sprintf(q, s.Table)
}

func (s *Sqlserver) ListMigrations() string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY id DESC`
	return fmt.Sprintf(q, s.Table)
}
