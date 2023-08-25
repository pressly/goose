package dialectquery

import "fmt"

type Ydb struct{}

var _ Querier = (*Ydb)(nil)

func (c *Ydb) CreateTable(tableName string) string {
	return fmt.Sprintf(`
		CREATE TABLE %s (
			hash Uint64,
			version_id Uint64,
			is_applied Bool,
			tstamp Timestamp,
	
			PRIMARY KEY(hash, version_id)
		);`,
		tableName,
	)
}

func (c *Ydb) InsertVersion(tableName string) string {
	return fmt.Sprintf(`
		UPSERT INTO %s (
			hash, 
			version_id, 
			is_applied, 
			tstamp
		) VALUES (
			Digest::IntHash64(CAST($1 AS Uint64)), 
			CAST($1 AS Uint64), 
			$2, 
			CurrentUtcTimestamp()
		);`,
		tableName,
	)
}

func (c *Ydb) DeleteVersion(tableName string) string {
	return fmt.Sprintf(`
		DELETE FROM %s 
		WHERE 
	    	hash = Digest::IntHash64(CAST($1 AS Uint64)) 
		AND 
		    version_id = $1;`,
		tableName,
	)
}

func (c *Ydb) GetMigrationByVersion(tableName string) string {
	return fmt.Sprintf(`
		SELECT tstamp, is_applied 
		FROM %s 
		WHERE 
		    hash = Digest::IntHash64(CAST($1 AS Uint64)) 
		AND 
		    version_id = $1 
		ORDER BY tstamp DESC LIMIT 1`,
		tableName,
	)
}

func (c *Ydb) ListMigrations(tableName string) string {
	return fmt.Sprintf(`
		SELECT version_id, is_applied 
		FROM %s 
		ORDER BY version_id DESC`,
		tableName,
	)
}
