package gooseutil

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveVersions(t *testing.T) {
	t.Run("not_allow_missing", func(t *testing.T) {
		// Nothing to apply nil
		got, err := UpVersions(nil, nil, math.MaxInt64, false)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))
		// Nothing to apply empty
		got, err = UpVersions([]int64{}, []int64{}, math.MaxInt64, false)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))

		// Nothing new
		got, err = UpVersions([]int64{1, 2, 3}, []int64{1, 2, 3}, math.MaxInt64, false)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))

		// All new
		got, err = UpVersions([]int64{1, 2, 3}, []int64{}, math.MaxInt64, false)
		require.NoError(t, err)
		require.Equal(t, 3, len(got))
		require.Equal(t, int64(1), got[0])
		require.Equal(t, int64(2), got[1])
		require.Equal(t, int64(3), got[2])

		// Squashed, no new
		got, err = UpVersions([]int64{3}, []int64{3}, math.MaxInt64, false)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))
		// Squashed, 1 new
		got, err = UpVersions([]int64{3, 4}, []int64{3}, math.MaxInt64, false)
		require.NoError(t, err)
		require.Equal(t, 1, len(got))
		require.Equal(t, int64(4), got[0])

		// Some new with target
		got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{1, 2}, 4, false)
		require.NoError(t, err)
		require.Equal(t, 2, len(got))
		require.Equal(t, int64(3), got[0])
		require.Equal(t, int64(4), got[1]) // up to and including target
		// Some new with zero target
		got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{1, 2}, 0, false)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))

		// Error: one missing migrations with max target
		_, err = UpVersions([]int64{1, 2, 3, 4}, []int64{1 /* 2*/, 3}, math.MaxInt64, false)
		require.Error(t, err)
		require.Equal(t,
			"detected 1 missing (out-of-order) migration lower than database version (3): version 2",
			err.Error(),
		)

		// Error: multiple missing migrations with max target
		_, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{ /* 1 */ 2 /* 3 */, 4, 5}, math.MaxInt64, false)
		require.Error(t, err)
		require.Equal(t,
			"detected 2 missing (out-of-order) migrations lower than database version (5): versions 1,3",
			err.Error(),
		)

		t.Run("target_lower_than_max", func(t *testing.T) {

			// These tests are a bit of an edge case but an important one worth documenting. There
			// can be missing migrations above and/or below the target version which itself can be
			// lower than the max db version. For example, migrations 1,2,3,4 in the filesystem, and
			// migrations 1,2,4 applied to the database and the user requested target 2. Technically
			// there are no missing migrations based on the target version since 1,2 have been
			// applied, but there is 1 missing migration (3) based on the max db version. Should
			// this return an error, or report no pending migrations?
			//
			// We've taken the stance that this SHOULD respect the target version and surface an
			// error if there are missing migrations below the target version. This is because the
			// user has explicitly requested a target version and we should respect that.

			got, err = UpVersions([]int64{1, 2, 3}, []int64{1 /* 2 */, 3}, 1, false)
			require.NoError(t, err)
			require.Equal(t, 0, len(got))
			got, err = UpVersions([]int64{1, 2, 3}, []int64{1 /* 2 */, 3}, 2, false)
			require.Error(t, err)
			require.Equal(t,
				"detected 1 missing (out-of-order) migration lower than database version (3), with target version (2): version 2",
				err.Error(),
			)
			got, err = UpVersions([]int64{1, 2, 3}, []int64{1 /* 2 */, 3}, 3, false)
			require.Error(t, err)
			require.Equal(t,
				"detected 1 missing (out-of-order) migration lower than database version (3), with target version (3): version 2",
				err.Error(),
			)

			_, err = UpVersions([]int64{1, 2, 3, 4, 5, 6}, []int64{1 /* 2 */, 3, 4 /* 5*/, 6}, 4, false)
			require.Error(t, err)
			require.Equal(t,
				"detected 1 missing (out-of-order) migration lower than database version (6), with target version (4): version 2",
				err.Error(),
			)
			_, err = UpVersions([]int64{1, 2, 3, 4, 5, 6}, []int64{1 /* 2 */, 3, 4 /* 5*/, 6}, 6, false)
			require.Error(t, err)
			require.Equal(t,
				"detected 2 missing (out-of-order) migrations lower than database version (6), with target version (6): versions 2,5",
				err.Error(),
			)
		})
	})

	t.Run("allow_missing", func(t *testing.T) {
		// Nothing to apply nil
		got, err := UpVersions(nil, nil, math.MaxInt64, true)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))
		// Nothing to apply empty
		got, err = UpVersions([]int64{}, []int64{}, math.MaxInt64, true)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))

		// Nothing new
		got, err = UpVersions([]int64{1, 2, 3}, []int64{1, 2, 3}, math.MaxInt64, true)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))

		// All new
		got, err = UpVersions([]int64{1, 2, 3}, []int64{}, math.MaxInt64, true)
		require.NoError(t, err)
		require.Equal(t, 3, len(got))
		require.Equal(t, int64(1), got[0])
		require.Equal(t, int64(2), got[1])
		require.Equal(t, int64(3), got[2])

		// Squashed, no new
		got, err = UpVersions([]int64{3}, []int64{3}, math.MaxInt64, true)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))
		// Squashed, 1 new
		got, err = UpVersions([]int64{3, 4}, []int64{3}, math.MaxInt64, true)
		require.NoError(t, err)
		require.Equal(t, 1, len(got))
		require.Equal(t, int64(4), got[0])

		// Some new with target
		got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{1, 2}, 4, true)
		require.NoError(t, err)
		require.Equal(t, 2, len(got))
		require.Equal(t, int64(3), got[0])
		require.Equal(t, int64(4), got[1]) // up to and including target
		// Some new with zero target
		got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{1, 2}, 0, true)
		require.NoError(t, err)
		require.Equal(t, 0, len(got))

		// No error: one missing
		got, err = UpVersions([]int64{1, 2, 3}, []int64{1 /* 2*/, 3}, math.MaxInt64, true)
		require.NoError(t, err)
		require.Equal(t, 1, len(got))
		require.Equal(t, int64(2), got[0]) // missing

		// No error: multiple missing and new with max target
		got, err = UpVersions([]int64{1, 2, 3, 4, 5}, []int64{ /* 1 */ 2 /* 3 */, 4}, math.MaxInt64, true)
		require.NoError(t, err)
		require.Equal(t, 3, len(got))
		require.Equal(t, int64(1), got[0]) // missing
		require.Equal(t, int64(3), got[1]) // missing
		require.Equal(t, int64(5), got[2])

		t.Run("target_lower_than_max", func(t *testing.T) {
			got, err = UpVersions([]int64{1, 2, 3}, []int64{1 /* 2 */, 3}, 1, true)
			require.NoError(t, err)
			require.Equal(t, 0, len(got))
			got, err = UpVersions([]int64{1, 2, 3}, []int64{1 /* 2 */, 3}, 2, true)
			require.NoError(t, err)
			require.Equal(t, 1, len(got))
			require.Equal(t, int64(2), got[0]) // missing
			got, err = UpVersions([]int64{1, 2, 3}, []int64{1 /* 2 */, 3}, 3, true)
			require.NoError(t, err)
			require.Equal(t, 1, len(got))
			require.Equal(t, int64(2), got[0]) // missing

			got, err = UpVersions([]int64{1, 2, 3, 4, 5, 6}, []int64{1 /* 2 */, 3, 4 /* 5*/, 6}, 4, true)
			require.NoError(t, err)
			require.Equal(t, 1, len(got))
			require.Equal(t, int64(2), got[0]) // missing
			got, err = UpVersions([]int64{1, 2, 3, 4, 5, 6}, []int64{1 /* 2 */, 3, 4 /* 5*/, 6}, 6, true)
			require.NoError(t, err)
			require.Equal(t, 2, len(got))
			require.Equal(t, int64(2), got[0]) // missing
			require.Equal(t, int64(5), got[1]) // missing
		})
	})

	t.Run("sort_ascending", func(t *testing.T) {
		got := []int64{5, 3, 4, 2, 1}
		sortAscending(got)
		require.Equal(t, []int64{1, 2, 3, 4, 5}, got)
	})
}
