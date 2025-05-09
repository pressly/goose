package goose_test

import (
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetDialect(t *testing.T) {
	tests := []struct {
		name string
		want goose.Dialect
	}{
		{
			name: "postgres",
			want: goose.DialectPostgres,
		},
		{
			name: "mysql",
			want: goose.DialectMySQL,
		},
		{
			name: "MySQL",
			want: goose.DialectMySQL,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dialect, err := goose.GetDialect(test.name)
			require.NoError(t, err)
			require.Equal(t, test.want, dialect)
		})
	}
}

func TestGetDialectFail(t *testing.T) {
	dialect, err := goose.GetDialect("fail")
	require.Empty(t, dialect)
	require.ErrorIs(t, err, goose.ErrUnknownDialect)
	require.EqualError(t, err, "fail: unknown dialect")
}
