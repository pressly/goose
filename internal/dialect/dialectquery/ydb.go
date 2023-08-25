package dialectquery

import "fmt"

type Ydb struct {
	Table string
}

var _ Querier = (*Ydb)(nil)

func (c *Ydb) CreateTable() string {
	return fmt.Sprintf(`
		CREATE TABLE %s (
			hash Uint64,
			version_id Uint64,
			is_applied Bool,
			tstamp Timestamp,
	
			PRIMARY KEY(hash, version_id)
		);`,
		c.Table,
	)
}

func (c *Ydb) InsertVersion() string {
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
		c.Table,
	)
}

func (c *Ydb) DeleteVersion() string {
	return fmt.Sprintf(`
		DELETE FROM %s 
		WHERE 
	    	hash = Digest::IntHash64(CAST($1 AS Uint64)) 
		AND 
		    version_id = $1;`,
		c.Table,
	)
}

func (c *Ydb) GetMigrationByVersion() string {
	return fmt.Sprintf(`
		SELECT tstamp, is_applied 
		FROM %s 
		WHERE 
		    hash = Digest::IntHash64(CAST($1 AS Uint64)) 
		AND 
		    version_id = $1 
		ORDER BY tstamp DESC LIMIT 1`,
		c.Table,
	)
}

func (c *Ydb) ListMigrations() string {
	return fmt.Sprintf(`
		SELECT version_id, is_applied 
		FROM %s 
		ORDER BY version_id DESC`,
		c.Table,
	)
}
