package database_test

import (
	"errors"
	"testing"

	"github.com/pressly/goose/v3/database"
)

func TestNewClickhouseReplicated_MissingClusterErrors(t *testing.T) {
	t.Setenv(database.EnvClickhouseCluster, "")

	_, err := database.NewClickhouseReplicated()
	if !errors.Is(err, database.ErrClickhouseReplicatedNoCluster) {
		t.Fatalf("expected ErrClickhouseReplicatedNoCluster, got: %v", err)
	}
}

func TestNewClickhouseReplicated_WithOptions(t *testing.T) {
	t.Setenv(database.EnvClickhouseCluster, "")

	q, err := database.NewClickhouseReplicated(database.WithClickhouseCluster("c1"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q == nil {
		t.Fatal("expected non-nil querier")
	}
	if got := q.CreateTable("goose_db_version"); got == "" {
		t.Fatal("expected non-empty CreateTable SQL")
	}
}

func TestNewStore_ClickhouseReplicated(t *testing.T) {
	t.Setenv(database.EnvClickhouseCluster, "envcluster")

	s, err := database.NewStore(database.DialectClickHouseReplicated, "goose_db_version")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil store")
	}
	if s.Tablename() != "goose_db_version" {
		t.Errorf("unexpected table name: %s", s.Tablename())
	}
}

func TestNewStore_ClickhouseReplicated_MissingClusterErrors(t *testing.T) {
	t.Setenv(database.EnvClickhouseCluster, "")

	_, err := database.NewStore(database.DialectClickHouseReplicated, "goose_db_version")
	if !errors.Is(err, database.ErrClickhouseReplicatedNoCluster) {
		t.Fatalf("expected ErrClickhouseReplicatedNoCluster, got: %v", err)
	}
}
