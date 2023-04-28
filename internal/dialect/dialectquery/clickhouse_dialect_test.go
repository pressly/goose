package dialectquery

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestClickhouseCreateTable(t *testing.T) {
	t.Parallel()

	type testData struct {
		clickhouse *Clickhouse
		result     string
	}

	tests := []testData{
		{
			clickhouse: &Clickhouse{
				Table: "schema_migrations",
				Params: clusterParameters{
					OnCluster:    true,
					ZooPath:      "/clickhouse/tables/{cluster}/{table}",
					ClusterMacro: "{cluster}",
					ReplicaMacro: "{replica}",
				},
			},
			result: `CREATE TABLE IF NOT EXISTS schema_migrations ON CLUSTER '{cluster}' (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime('UTC') default now()
	)
    ENGINE = ReplicatedMergeTree('/clickhouse/tables/{cluster}/{table}', '{replica}')
	ORDER BY (date)`,
		},
		{
			clickhouse: &Clickhouse{
				Table: "schema_migrations_v1",
				Params: clusterParameters{
					OnCluster:    true,
					ZooPath:      "/clickhouse/tables/dev-cluster/{table}",
					ClusterMacro: "dev-cluster",
					ReplicaMacro: "{replica}",
				},
			},
			result: `CREATE TABLE IF NOT EXISTS schema_migrations_v1 ON CLUSTER 'dev-cluster' (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime('UTC') default now()
	)
    ENGINE = ReplicatedMergeTree('/clickhouse/tables/dev-cluster/{table}', '{replica}')
	ORDER BY (date)`,
		},
	}

	for _, test := range tests {
		out := test.clickhouse.CreateTable()
		if diff := cmp.Diff(test.result, out); diff != "" {
			t.Errorf("clickhouse.CreateTable() mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestClickhouseAttachOptions(t *testing.T) {
	t.Parallel()

	type testData struct {
		options  map[string]string
		input    *Clickhouse
		err      error
		expected clusterParameters
	}

	tests := []testData{
		{
			options: map[string]string{
				"ON_CLUSTER": "true",
			},
			input: &Clickhouse{},
			err:   nil,
			expected: clusterParameters{
				OnCluster:    true,
				ZooPath:      "/clickhouse/tables/{cluster}/{table}",
				ClusterMacro: "{cluster}",
				ReplicaMacro: "{replica}",
			},
		},
		{
			options: map[string]string{
				"ON_CLUSTER": "true",
				"ZOO_PATH":   "/clickhouse/hard_coded_path",
			},
			input: &Clickhouse{},
			err:   nil,
			expected: clusterParameters{
				OnCluster:    true,
				ZooPath:      "/clickhouse/hard_coded_path",
				ClusterMacro: "{cluster}",
				ReplicaMacro: "{replica}",
			},
		},
		{
			options: map[string]string{
				"ON_CLUSTER":    "true",
				"ZOO_PATH":      "/clickhouse/hard_coded_path",
				"CLUSTER_MACRO": "dev-cluster",
				"REPLICA_MACRO": "replica-01",
			},
			input: &Clickhouse{},
			err:   nil,
			expected: clusterParameters{
				OnCluster:    true,
				ZooPath:      "/clickhouse/hard_coded_path",
				ClusterMacro: "dev-cluster",
				ReplicaMacro: "replica-01",
			},
		},
		{
			options: map[string]string{
				"ON_CLUSTER": "false",
			},
			input: &Clickhouse{},
			err:   nil,
			expected: clusterParameters{
				OnCluster: false,
			},
		},
	}

	for _, test := range tests {
		err := test.input.AttachOptions(test.options)
		if err != test.err {
			t.Errorf("AttachOptions mismatch expected error: %v, got: %v", test.err, err)
		}
		if diff := cmp.Diff(test.expected, test.input.Params); diff != "" {
			t.Errorf("clickhouse.AttachOptions() mismatch (-want +got):\n%s", diff)
		}
	}

}
