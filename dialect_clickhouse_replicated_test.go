package goose_test

import (
	"testing"

	"github.com/pressly/goose/v3"
)

func TestSetDialect_ClickhouseReplicated(t *testing.T) {
	t.Setenv(goose.EnvClickhouseCluster, "envcluster")

	if err := goose.SetDialect("clickhouse-replicated"); err != nil {
		t.Fatalf("SetDialect returned error: %v", err)
	}
}

func TestSetDialect_ClickhouseReplicated_MissingClusterErrors(t *testing.T) {
	t.Setenv(goose.EnvClickhouseCluster, "")

	err := goose.SetDialect("clickhouse-replicated")
	if err == nil {
		t.Fatal("expected error when GOOSE_CLICKHOUSE_CLUSTER is unset")
	}
}
