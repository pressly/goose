package dialect_test

import (
	"github.com/pressly/goose/v3/internal/dialect"
	"github.com/stretchr/testify/require"
	"testing"
)

var _testUnmarshalData = []struct {
	name string
	want dialect.Dialect
}{
	{
		name: "postgres",
		want: dialect.Postgres,
	},
	{
		name: "mysql",
		want: dialect.Mysql,
	},
	{
		name: "MySQL",
		want: dialect.Mysql,
	},
}

func TestDialect_GetDialect(t *testing.T) {
	for _, test := range _testUnmarshalData {
		t.Run(test.name, func(t *testing.T) {
			d, err := dialect.GetDialect(test.name)
			require.NoError(t, err)
			require.Equal(t, test.want, d)
		})
	}
}

func TestDialect_GetDialectFail(t *testing.T) {
	d, err := dialect.GetDialect("fail")
	require.Empty(t, d)
	require.ErrorIs(t, err, dialect.ErrUnknownDialect)
	require.EqualError(t, err, "fail: unknown dialect")
}

func TestDialect_UnmarshalText(t *testing.T) {
	for _, test := range _testUnmarshalData {
		t.Run(test.name, func(t *testing.T) {
			var d dialect.Dialect
			require.NoError(t, d.UnmarshalText([]byte(test.name)))
		})
	}
}

func TestDialect_UnmarshalTextFail(t *testing.T) {
	var d dialect.Dialect
	var err = d.UnmarshalText([]byte("fail"))
	require.ErrorIs(t, err, dialect.ErrUnknownDialect)
	require.EqualError(t, err, "fail: unknown dialect")
}
