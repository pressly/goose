package testdb

import (
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

// Cluster name declared inside the bind-mounted ch1.xml / ch2.xml — see
// internal/testing/integration/clickhouse-replicated/config/. Kept in sync
// with those files.
const CLICKHOUSE_REPLICATED_CLUSTER = "goose_cluster"

// NewClickHouseReplicated brings up a two-node replicated ClickHouse cluster
// (`ch1`, `ch2`) with an embedded Keeper on ch1, using ory/dockertest. The
// existing config XMLs under
// internal/testing/integration/clickhouse-replicated/config/ are bind-mounted
// into each container read-only, so the cluster topology is identical to the
// (deleted) docker-compose setup.
//
// Returns a *sql.DB connected to ch1, a second *sql.DB connected to ch2 (used
// by tests that verify replication), and a cleanup that purges both
// containers and removes the user network.
func NewClickHouseReplicated(opts ...OptionsFunc) (ch1DB, ch2DB *sql.DB, cleanup func(), err error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, err
	}

	configDir, err := clickHouseReplicatedConfigDir()
	if err != nil {
		return nil, nil, nil, err
	}

	network, err := pool.CreateNetwork("goose-ch-repl-" + randSuffix())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("create network: %w", err)
	}

	// Track resources for the cleanup closure so partial failure paths still
	// tear down.
	var containers []*dockertest.Resource
	cleanup = func() {
		if option.debug {
			return
		}
		for _, c := range containers {
			if err := pool.Purge(c); err != nil {
				log.Printf("failed to purge resource: %v", err)
			}
		}
		if err := network.Close(); err != nil {
			log.Printf("failed to close network: %v", err)
		}
	}
	// If we bail before returning success, tear down what we brought up.
	success := false
	defer func() {
		if !success {
			cleanup()
		}
	}()

	ch1, err := runClickHouseReplicatedNode(pool, network, configDir, "ch1", "1")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("start ch1: %w", err)
	}
	containers = append(containers, ch1)

	ch2, err := runClickHouseReplicatedNode(pool, network, configDir, "ch2", "1")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("start ch2: %w", err)
	}
	containers = append(containers, ch2)

	ch1Addr := "localhost:" + ch1.GetPort("9000/tcp")
	ch2Addr := "localhost:" + ch2.GetPort("9000/tcp")

	if err := pool.Retry(func() error {
		ch1DB = clickHouseReplicatedOpen(ch1Addr)
		return ch1DB.Ping()
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("connect to ch1 (%s): %w", ch1Addr, err)
	}
	if err := pool.Retry(func() error {
		ch2DB = clickHouseReplicatedOpen(ch2Addr)
		return ch2DB.Ping()
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("connect to ch2 (%s): %w", ch2Addr, err)
	}

	// Ping only proves the CH TCP socket is up. `ON CLUSTER` DDL additionally
	// needs Keeper reachable and both replicas registered against the cluster
	// definition. Gate on that here so goose's `CREATE TABLE ... ON CLUSTER`
	// during Provider.Init doesn't race with cluster bootstrap.
	if err := pool.Retry(func() error {
		return clickHouseReplicatedClusterReady(ch1DB)
	}); err != nil {
		return nil, nil, nil, fmt.Errorf("cluster %q not ready: %w", CLICKHOUSE_REPLICATED_CLUSTER, err)
	}

	success = true
	return ch1DB, ch2DB, cleanup, nil
}

// clickHouseReplicatedClusterReady returns nil once ch1 can see both replicas
// of goose_cluster and its Keeper root znode is reachable. That is a
// necessary (and effectively sufficient) precondition for `ON CLUSTER` DDL to
// succeed without racing.
func clickHouseReplicatedClusterReady(db *sql.DB) error {
	var hosts int
	if err := db.QueryRow(
		`SELECT count() FROM system.clusters WHERE cluster = ?`,
		CLICKHOUSE_REPLICATED_CLUSTER,
	).Scan(&hosts); err != nil {
		return fmt.Errorf("query system.clusters: %w", err)
	}
	if hosts < 2 {
		return fmt.Errorf("cluster %q has %d hosts, want 2", CLICKHOUSE_REPLICATED_CLUSTER, hosts)
	}
	// Probe Keeper by reading the root znode; fails cleanly until keeper is up.
	var zk int
	if err := db.QueryRow(`SELECT count() FROM system.zookeeper WHERE path = '/'`).Scan(&zk); err != nil {
		return fmt.Errorf("query system.zookeeper: %w", err)
	}
	return nil
}

func runClickHouseReplicatedNode(
	pool *dockertest.Pool,
	network *dockertest.Network,
	configDir, name, shard string,
) (*dockertest.Resource, error) {
	// ch1 runs the embedded keeper; ch2 is a plain replica that points at
	// keeper on ch1. Both share the same cluster.xml layout modulo hostnames.
	clusterXML := filepath.Join(configDir, name+".xml")
	macrosXML := filepath.Join(configDir, "macros-"+name+".xml")
	keeperXML := filepath.Join(configDir, "keeper.xml")
	if name != "ch1" {
		keeperXML = filepath.Join(configDir, "keeper-client.xml")
	}
	usersXML := filepath.Join(configDir, "users.xml")

	binds := []string{
		clusterXML + ":/etc/clickhouse-server/config.d/cluster.xml:ro",
		macrosXML + ":/etc/clickhouse-server/config.d/macros.xml:ro",
		keeperXML + ":/etc/clickhouse-server/config.d/keeper.xml:ro",
		usersXML + ":/etc/clickhouse-server/users.d/users.xml:ro",
	}

	runOptions := &dockertest.RunOptions{
		Repository: CLICKHOUSE_IMAGE,
		Tag:        "24.8",
		Hostname:   name,
		Name:       "goose-ch-repl-" + name + "-" + randSuffix(),
		NetworkID:  network.Network.ID,
		Env: []string{
			"CLICKHOUSE_SHARD=" + shard,
			"CLICKHOUSE_REPLICA=" + name,
		},
		Labels:       map[string]string{"goose_test": "1"},
		ExposedPorts: []string{"9000/tcp"},
	}

	return pool.RunWithOptions(runOptions, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		config.Binds = binds
		config.Ulimits = []docker.ULimit{
			{Name: "nofile", Soft: 262144, Hard: 262144},
		}
	})
}

func clickHouseReplicatedOpen(addr string) *sql.DB {
	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(10)
	db.SetConnMaxLifetime(time.Hour)
	return db
}

// clickHouseReplicatedConfigDir returns the absolute path of the shared
// cluster config directory, resolved relative to this source file so tests
// can be run from any working directory.
func clickHouseReplicatedConfigDir() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	// internal/testing/testdb/clickhouse_replicated.go →
	// internal/testing/integration/clickhouse-replicated/config
	dir := filepath.Join(
		filepath.Dir(thisFile),
		"..", "integration", "clickhouse-replicated", "config",
	)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func randSuffix() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
