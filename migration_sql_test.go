package goose

import (
	"os"
	"strings"
	"testing"
)

func TestSemicolons(t *testing.T) {

	type testData struct {
		line   string
		result bool
	}

	tests := []testData{
		{
			line:   "END;",
			result: true,
		},
		{
			line:   "END; -- comment",
			result: true,
		},
		{
			line:   "END   ; -- comment",
			result: true,
		},
		{
			line:   "END -- comment",
			result: false,
		},
		{
			line:   "END -- comment ;",
			result: false,
		},
		{
			line:   "END \" ; \" -- comment",
			result: false,
		},
	}

	for _, test := range tests {
		r := endsWithSemicolon(test.line)
		if r != test.result {
			t.Errorf("incorrect semicolon. got %v, want %v", r, test.result)
		}
	}
}

func TestSplitStatements(t *testing.T) {

	type testData struct {
		sql       string
		direction bool
		count     int
	}

	tests := []testData{
		{
			sql:       functxt,
			direction: true,
			count:     2,
		},
		{
			sql:       functxt,
			direction: false,
			count:     2,
		},
		{
			sql:       multitxt,
			direction: true,
			count:     2,
		},
		{
			sql:       multitxt,
			direction: false,
			count:     2,
		},
	}

	for _, test := range tests {
		stmts, _, err := getSQLStatements(strings.NewReader(test.sql), test.direction)
		if err != nil {
			t.Error(err)
		}
		if len(stmts) != test.count {
			t.Errorf("incorrect number of stmts. got %v, want %v", len(stmts), test.count)
		}
	}
}

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
		_, useTx, err := getSQLStatements(f, true)
		if err != nil {
			t.Error(err)
		}
		if useTx != test.useTransactions {
			t.Errorf("Failed transaction check. got %v, want %v", useTx, test.useTransactions)
		}
		f.Close()
	}
}

func TestParsingErrors(t *testing.T) {
	type testData struct {
		sql   string
		error bool
	}
	tests := []testData{
		{
			sql:   statementBeginNoStatementEnd,
			error: true,
		},
		{
			sql:   unfinishedSQL,
			error: true,
		},
		{
			sql:   noUpDownAnnotations,
			error: true,
		},
	}
	for _, test := range tests {
		_, _, err := getSQLStatements(strings.NewReader(test.sql), true)
		if err == nil {
			t.Errorf("Failed transaction check. got %v, want %v", err, test.error)
		}
	}
}

var functxt = `-- +goose Up
CREATE TABLE IF NOT EXISTS histories (
  id                BIGSERIAL  PRIMARY KEY,
  current_value     varchar(2000) NOT NULL,
  created_at      timestamp with time zone  NOT NULL
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
BEGIN
  FOR create_query IN SELECT
      'CREATE TABLE IF NOT EXISTS histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( CHECK( created_at >= timestamp '''
      || TO_CHAR( d, 'YYYY-MM-DD 00:00:00' )
      || ''' AND created_at < timestamp '''
      || TO_CHAR( d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00' )
      || ''' ) ) inherits ( histories );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
  END LOOP;  -- LOOP END
END;         -- FUNCTION END
$$
language plpgsql;
-- +goose StatementEnd

-- +goose Down
drop function histories_partition_creation(DATE, DATE);
drop TABLE histories;
`

// test multiple up/down transitions in a single script
var multitxt = `-- +goose Up
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE post;

-- +goose Up
CREATE TABLE fancier_post (
    id int NOT NULL,
    title text,
    body text,
    created_on timestamp without time zone,
    PRIMARY KEY(id)
);

-- +goose Down
DROP TABLE fancier_post;
`

var statementBeginNoStatementEnd = `-- +goose Up
CREATE TABLE IF NOT EXISTS histories (
  id                BIGSERIAL  PRIMARY KEY,
  current_value     varchar(2000) NOT NULL,
  created_at      timestamp with time zone  NOT NULL
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION histories_partition_creation( DATE, DATE )
returns void AS $$
DECLARE
  create_query text;
BEGIN
  FOR create_query IN SELECT
      'CREATE TABLE IF NOT EXISTS histories_'
      || TO_CHAR( d, 'YYYY_MM' )
      || ' ( CHECK( created_at >= timestamp '''
      || TO_CHAR( d, 'YYYY-MM-DD 00:00:00' )
      || ''' AND created_at < timestamp '''
      || TO_CHAR( d + INTERVAL '1 month', 'YYYY-MM-DD 00:00:00' )
      || ''' ) ) inherits ( histories );'
    FROM generate_series( $1, $2, '1 month' ) AS d
  LOOP
    EXECUTE create_query;
  END LOOP;  -- LOOP END
END;         -- FUNCTION END
$$
language plpgsql;

-- +goose Down
drop function histories_partition_creation(DATE, DATE);
drop TABLE histories;
`

var unfinishedSQL = `
-- +goose Up
ALTER TABLE post

-- +goose Down
`
var noUpDownAnnotations = `
CREATE TABLE post (
    id int NOT NULL,
    title text,
    body text,
    PRIMARY KEY(id)
);
`
