package goose

import (
	"os"
	"testing"
)

func TestUseTransactions(t *testing.T) {
	type testData struct {
		fileName        string
		useTransactions bool
	}

	tests := []testData{
		{
			fileName:        "./examples/sql-migrations/00001_create_users_table.sql",
			useTransactions: true,
		},
		{
			fileName:        "./examples/sql-migrations/00002_rename_root.sql",
			useTransactions: true,
		},
		{
			fileName:        "./examples/sql-migrations/00003_no_transaction.sql",
			useTransactions: false,
		},
	}

	for _, test := range tests {
		f, err := os.Open(test.fileName)
		if err != nil {
			t.Error(err)
		}
		_, useTx := getSQLQuery(f, true)
		if useTx != test.useTransactions {
			t.Errorf("Failed transaction check. got %v, want %v", useTx, test.useTransactions)
		}
		f.Close()
	}
}
