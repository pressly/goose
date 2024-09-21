package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/sethvargo/go-retry"
)

const clickhouseReplicatedNetworkName = "goose-clickhouse-replicated-tests"

func newClickHouseReplicated(opts ...OptionsFunc) (*sql.DB, *sql.DB, func(), error) {
	option := &options{}
	for _, f := range opts {
		f(option)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, nil, nil, err
	}

	net, err := pool.CreateNetwork(clickhouseReplicatedNetworkName)
	if err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return nil, nil, nil, err
		}

		nets, err := pool.NetworksByName(clickhouseReplicatedNetworkName)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(nets) != 1 {
			return nil, nil, nil, fmt.Errorf("found more than one network with name %s", clickhouseReplicatedNetworkName)
		}

		net = &nets[0]
	}

	zk, err := startZookeeper(pool, net)
	if err != nil {
		return nil, nil, nil, err
	}

	path, err := filepath.Abs("./../../../")
	if err != nil {
		return nil, nil, nil, err
	}

	db0, cleanup0, err := NewClickHouse(
		WithName("clickhouse0"),
		WithEnv([]string{"REPLICA_NAME=clickhouse0"}),
		WithNetwork(net),
		WithMounts([]string{
			path + "/testdata/clickhouse-replicated:/etc/clickhouse-server/conf.d",
		}))
	if err != nil {
		return nil, nil, nil, err
	}

	db1, cleanup1, err := NewClickHouse(
		WithName("clickhouse1"),
		WithEnv([]string{"REPLICA_NAME=clickhouse1"}),
		WithNetwork(net),
		WithMounts([]string{
			path + "/testdata/clickhouse-replicated:/etc/clickhouse-server/conf.d",
		}))
	if err != nil {
		return nil, nil, nil, err
	}

	cleanup := func() {
		if option.debug {
			return
		}

		cleanup0()
		cleanup1()
		if err := pool.Purge(zk); err != nil {
			log.Printf("failed to purge resource: %v", err)
		}
		if err := pool.RemoveNetwork(net); err != nil {
			log.Printf("failed to purge network %s: %v", net.Network.Name, err)
		}
	}

	return db0, db1, cleanup, nil
}

func startZookeeper(pool *dockertest.Pool, net *dockertest.Network) (*dockertest.Resource, error) {
	runOptions := &dockertest.RunOptions{
		Name:         "zookeeper",
		Repository:   "zookeeper",
		Tag:          "3.7.2",
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
		NetworkID:    net.Network.ID,
	}
	zk, err := pool.RunWithOptions(
		runOptions,
		func(config *docker.HostConfig) {
			// Set AutoRemove to true so that stopped container goes away by itself.
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return nil, err
	}

	backoff := retry.WithMaxDuration(1*time.Minute, retry.NewConstant(2*time.Second))
	if err := retry.Do(context.Background(), backoff, func(ctx context.Context) error {
		exitCode, err := zk.Exec([]string{"zkCli.sh", "ls", "/"}, dockertest.ExecOptions{})
		if err != nil {
			return retry.RetryableError(err)
		}
		if exitCode != 0 {
			return retry.RetryableError(fmt.Errorf("zk cmd returns %d", exitCode))
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("could not connect to docker database: %v", err)
	}

	return zk, nil
}
