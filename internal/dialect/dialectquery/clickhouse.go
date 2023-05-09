package dialectquery

import "fmt"

const (
	paramOnCluster    = "ON_CLUSTER"
	paramClusterMacro = "CLUSTER_MACRO"
)

type clusterParameters struct {
	OnCluster    bool
	ClusterMacro string
}

type Clickhouse struct {
	Table  string
	Params clusterParameters
}

var _ Querier = (*Clickhouse)(nil)

func (c *Clickhouse) CreateTable(tableName string) string {
	q := `CREATE TABLE IF NOT EXISTS %s (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime64(9, 'UTC') default now64(9, 'UTC')
	  )
	  ENGINE = KeeperMap('/goose_version')
	  PRIMARY KEY version_id`

	qCluster := `CREATE TABLE IF NOT EXISTS %s ON CLUSTER '%s' (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime64(9, 'UTC') default now64(9, 'UTC')
	)
    ENGINE = KeeperMap('/goose_version_repl')
	PRIMARY KEY version_id`

	if c.Params.OnCluster {
		return fmt.Sprintf(qCluster, tableName, c.Params.ClusterMacro)
	}
	return fmt.Sprintf(q, tableName)
}

func (c *Clickhouse) InsertVersion(tableName string) string {
	q := `INSERT INTO %s (version_id, is_applied) VALUES ($1, $2)`
	return fmt.Sprintf(q, tableName)
}

func (c *Clickhouse) DeleteVersion(tableName string) string {
	q := `ALTER TABLE %s DELETE WHERE version_id = $1 SETTINGS mutations_sync = 2`
	return fmt.Sprintf(q, tableName)
}

func (c *Clickhouse) GetMigrationByVersion(tableName string) string {
	q := `SELECT tstamp, is_applied FROM %s WHERE version_id = $1 ORDER BY tstamp DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (c *Clickhouse) ListMigrations(tableName string) string {
	q := `SELECT version_id, is_applied FROM %s ORDER BY tstamp DESC`
	return fmt.Sprintf(q, tableName)
}

func (c *Clickhouse) AttachOptions(options map[string]string) error {
	if val, ok := options[paramOnCluster]; ok {
		if val == "true" {
			clusterMacro, ok := options[paramClusterMacro]
			if !ok {
				clusterMacro = "{cluster}"
			}
			c.Params.ClusterMacro = clusterMacro
			c.Params.OnCluster = true
		}
	}
	return nil
}
