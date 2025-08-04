package gooseutil

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type MockSQLResult struct {
	MockLastInsertId func() (int64, error)
	MockRowsAffected func() (int64, error)
}

func (mr MockSQLResult) RowsAffected() (int64, error) {
	return mr.MockRowsAffected()
}

func (mr MockSQLResult) LastInsertId() (int64, error) {
	return mr.MockLastInsertId()
}

func TestFormatSQLResultInfo(t *testing.T) {
	t.Parallel()

	// Base mocks
	nothingSupportedRes := MockSQLResult{
		MockRowsAffected: func() (int64, error) {
			return 0, errors.New("dummy error")
		},
		MockLastInsertId: func() (int64, error) {
			return 0, errors.New("dummy error")
		},
	}
	bothSupportedRes := MockSQLResult{
		MockRowsAffected: func() (int64, error) { return 1, nil },
		MockLastInsertId: func() (int64, error) { return 2, nil },
	}

	// Nothing supported
	got := FormatSQLResultInfo(nothingSupportedRes)
	require.Equal(t, "", got)

	// Both supported
	got = FormatSQLResultInfo(bothSupportedRes)
	require.Equal(t, "rows affected: 1, last insert id: 2", got)

	// Only RowsAffected supported
	rowsAffectedSupportedRes := nothingSupportedRes
	rowsAffectedSupportedRes.MockRowsAffected = bothSupportedRes.MockRowsAffected
	got = FormatSQLResultInfo(rowsAffectedSupportedRes)
	require.Equal(t, "rows affected: 1", got)

	// Only LastInsertId supported
	lastInsertIdSupportedRes := nothingSupportedRes
	lastInsertIdSupportedRes.MockLastInsertId = bothSupportedRes.MockLastInsertId
	got = FormatSQLResultInfo(lastInsertIdSupportedRes)
	require.Equal(t, "last insert id: 2", got)
}
