package dialectquery

import "fmt"

const (
	paramOnCluster    = "ON_CLUSTER"
	paramZooPath      = "ZOO_PATH"
	paramClusterMacro = "CLUSTER_MACRO"
	paramReplicaMacro = "REPLICA_MACRO"
)

type clusterParameters struct {
	OnCluster    bool
	ZooPath      string
	ClusterMacro string
	ReplicaMacro string
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
		tstamp DateTime default now()
	  )
	  ENGINE = MergeTree()
		ORDER BY (date)`

	qCluster := `CREATE TABLE IF NOT EXISTS %s ON CLUSTER '%s' (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime('UTC') default now()
	)
    ENGINE = ReplicatedMergeTree('%s', '%s')
	ORDER BY (date)`

	if c.Params.OnCluster {
		return fmt.Sprintf(qCluster, c.Table, c.Params.ClusterMacro, c.Params.ZooPath, c.Params.ReplicaMacro)
	}
	return fmt.Sprintf(q, c.Table)
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
	q := `SELECT version_id, is_applied FROM %s ORDER BY version_id DESC`
	return fmt.Sprintf(q, tableName)
}

func (c *Clickhouse) AttachOptions(options map[string]string) error {
	if val, ok := options[paramOnCluster]; ok {
		if val == "true" {
			clusterMacro, ok := options[paramClusterMacro]
			if !ok {
				clusterMacro = "{cluster}"
			}
			zooPath, ok := options[paramZooPath]
			if !ok {
				zooPath = fmt.Sprintf("/clickhouse/tables/%s/{table}", clusterMacro)
			}
			replicaMacro, ok := options[paramReplicaMacro]
			if !ok {
				replicaMacro = "{replica}"
			}
			c.Params.ZooPath = zooPath
			c.Params.ClusterMacro = clusterMacro
			c.Params.ReplicaMacro = replicaMacro
			c.Params.OnCluster = true
		}
	}
	return nil
}
