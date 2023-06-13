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
				Params: clusterParameters{
					OnCluster:    true,
					ClusterMacro: "{cluster}",
				},
			},
			result: `CREATE TABLE IF NOT EXISTS schema_migrations ON CLUSTER '{cluster}' (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime64(9, 'UTC') default now64(9, 'UTC')
	)
    ENGINE = KeeperMap('/goose_version_repl')
	PRIMARY KEY version_id`,
		},
		{
			clickhouse: &Clickhouse{
				Params: clusterParameters{
					OnCluster:    true,
					ClusterMacro: "dev-cluster",
				},
			},
			result: `CREATE TABLE IF NOT EXISTS schema_migrations ON CLUSTER 'dev-cluster' (
		version_id Int64,
		is_applied UInt8,
		date Date default now(),
		tstamp DateTime64(9, 'UTC') default now64(9, 'UTC')
	)
    ENGINE = KeeperMap('/goose_version_repl')
	PRIMARY KEY version_id`,
		},
	}

	for _, test := range tests {
		out := test.clickhouse.CreateTable("schema_migrations")
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
				ClusterMacro: "{cluster}",
			},
		},
		{
			options: map[string]string{
				"ON_CLUSTER":    "true",
				"CLUSTER_MACRO": "dev-cluster",
			},
			input: &Clickhouse{},
			err:   nil,
			expected: clusterParameters{
				OnCluster:    true,
				ClusterMacro: "dev-cluster",
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
