package database

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPlaceholderFormatRewrite(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format PlaceholderFormat
		query  string
		want   string
	}{
		{
			name:   "default",
			format: PlaceholderDefault,
			query:  "SELECT * FROM goose_db_version WHERE version_id=$1",
			want:   "SELECT * FROM goose_db_version WHERE version_id=$1",
		},
		{
			name:   "question from dollar",
			format: PlaceholderQuestion,
			query:  "INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, $2)",
			want:   "INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, ?)",
		},
		{
			name:   "question from sqlserver",
			format: PlaceholderQuestion,
			query:  "DELETE FROM goose_db_version WHERE version_id=@p1",
			want:   "DELETE FROM goose_db_version WHERE version_id=?",
		},
		{
			name:   "dollar",
			format: PlaceholderDollar,
			query:  "INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, ?)",
			want:   "INSERT INTO goose_db_version (version_id, is_applied) VALUES ($1, $2)",
		},
		{
			name:   "sqlserver",
			format: PlaceholderAtP,
			query:  "INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, ?)",
			want:   "INSERT INTO goose_db_version (version_id, is_applied) VALUES (@p1, @p2)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.format.rewrite(tt.query)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
