package dialectquery

import "fmt"

type Sqlserver struct{}

var _ Querier = (*Sqlserver)(nil)

func (s *Sqlserver) CreateTable(tableName string) string {
	q := `CREATE TABLE %s (
		id INT NOT NULL IDENTITY(1,1) PRIMARY KEY,
		version_id BIGINT NOT NULL,
		is_applied BIT NOT NULL,
		tstamp DATETIME NULL DEFAULT CURRENT_TIMESTAMP
	)`
	return fmt.Sprintf(q, tableName)
}

func (s *Sqlserver) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES (@p1, @p2)`
	return fmt.Sprintf(q, tableName)
}

func (s *Sqlserver) DeleteVersion(tableName string) string {
	q := `DELETE FROM %s WHERE version_id=@p1`
	return fmt.Sprintf(q, tableName)
}

func (s *Sqlserver) GetMigrationByVersion(tableName string) string {
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
	return fmt.Sprintf(q, tableName)
}

func (s *Sqlserver) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY id DESC`
	return fmt.Sprintf(q, tableName)
}
