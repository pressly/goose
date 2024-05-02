package gooseutil

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveVersions(t *testing.T) {
	got, err := UpVersions(nil, nil, 0, false)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))

	// Nothing to apply
	got, err = UpVersions([]int64{1, 2, 3}, []int64{1, 2, 3}, math.MaxInt64, false)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))
	// Nothing to apply with missing allowed
	got, err = UpVersions([]int64{1, 2, 3}, []int64{1, 2, 3}, math.MaxInt64, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))

	// All new
	got, err = UpVersions([]int64{1, 2, 3}, []int64{}, math.MaxInt64, false)
	require.NoError(t, err)
	require.Equal(t, 3, len(got))
	require.Equal(t, int64(1), got[0])
	require.Equal(t, int64(2), got[1])
	require.Equal(t, int64(3), got[2])

	// Squashed, all old
	got, err = UpVersions([]int64{3}, []int64{3}, math.MaxInt64, false)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))

	// New migrations with missing not allowed
	got, err = UpVersions([]int64{1, 2, 3}, []int64{1, 2}, math.MaxInt64, false)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, int64(3), got[0])
	// New migrations with missing allowed
	got, err = UpVersions([]int64{1, 2, 3}, []int64{1, 2}, math.MaxInt64, true)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, int64(3), got[0])

	// One missing migration with missing allowed
	got, err = UpVersions([]int64{1, 2, 3}, []int64{1, 3}, math.MaxInt64, true)
	require.NoError(t, err)
	require.Equal(t, 1, len(got))
	require.Equal(t, int64(2), got[0])
	// Multiple missing migrations with missing allowed
	got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{2, 4, 5}, math.MaxInt64, true)
	require.NoError(t, err)
	require.Equal(t, 2, len(got))
	require.Equal(t, int64(1), got[0])
	require.Equal(t, int64(3), got[1])
	// Multiple missing migrations and new with missing allowed
	got, err = UpVersions([]int64{1, 2, 3, 4, 5, 6}, []int64{2, 4, 5}, math.MaxInt64, true)
	require.NoError(t, err)
	require.Equal(t, 3, len(got))
	require.Equal(t, int64(1), got[0])
	require.Equal(t, int64(3), got[1])
	require.Equal(t, int64(6), got[2])

	// Missing migrations, no new, with missing not allowed
	_, err = UpVersions([]int64{1, 2, 3, 4}, []int64{1, 4}, math.MaxInt64, false)
	require.Error(t, err)
	require.Equal(t,
		"found 2 missing (out-of-order) migrations lower than current database max version (4): versions 2,3",
		err.Error(),
	)
	// One missing migration and one new with missing not allowed
	_, err = UpVersions([]int64{1, 2, 3, 4}, []int64{1, 3}, math.MaxInt64, false)
	require.Error(t, err)
	require.Equal(t,
		"found 1 missing (out-of-order) migration lower than current database max version (3): version 2",
		err.Error(),
	)

	// Missing multiple migrations with one new missing not allowed
	_, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{1, 4}, math.MaxInt64, false)
	require.Error(t, err)
	require.Equal(t,
		"found 2 missing (out-of-order) migrations lower than current database max version (4): versions 2,3",
		err.Error(),
	)

	// Squashed migrations
	got, err = UpVersions([]int64{5}, []int64{1, 2, 3, 4, 5}, 3, false)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))
	got, err = UpVersions([]int64{5}, []int64{1, 2, 3, 4, 5}, 3, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))

	// With target version
	got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{1, 2, 3}, 3, false)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))
	got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{1, 2, 3}, 3, true)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))

	// With target version
	got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{1}, 4, false)
	require.NoError(t, err)
	require.Equal(t, 3, len(got))
	require.Equal(t, int64(2), got[0])
	require.Equal(t, int64(3), got[1])
	require.Equal(t, int64(4), got[2])

	// no input ordering guarantees, but we can check that the output is sorted
	got, err = UpVersions([]int64{5, 4, 3, 2, 1}, []int64{2, 1, 3}, 3, false)
	require.NoError(t, err)
	require.Equal(t, 0, len(got))
	// same, but with a max target
	got, err = UpVersions([]int64{5, 4, 3, 2, 1}, []int64{2, 1, 3}, math.MaxInt64, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(got))
	require.Equal(t, int64(4), got[0])
	require.Equal(t, int64(5), got[1])
	// same, but with missing allowed
	got, err = UpVersions([]int64{5, 4, 3, 2, 1}, []int64{1, 3}, math.MaxInt64, true)
	require.NoError(t, err)
	require.Equal(t, 3, len(got))
	require.Equal(t, int64(2), got[0])
	require.Equal(t, int64(4), got[1])
	require.Equal(t, int64(5), got[2])
	// same, but with missing not allowed
	_, err = UpVersions([]int64{5, 4, 3, 2, 1}, []int64{1, 3}, math.MaxInt64, false)
	require.Error(t, err)
	require.Equal(t,
		"found 1 missing (out-of-order) migration lower than current database max version (3): version 2",
		err.Error(),
	)

	t.Run("sort_ascending", func(t *testing.T) {
		got := []int64{5, 3, 4, 2, 1}
		sortAscending(got)
		require.Equal(t, []int64{1, 2, 3, 4, 5}, got)
	})
}
