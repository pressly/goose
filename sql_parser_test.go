package goose

import (
	"os"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func TestSemicolons(t *testing.T) {
	t.Parallel()

	type testData struct {
		line   string
		result bool
	}

	tests := []testData{
		{line: "END;", result: true},
		{line: "END; -- comment", result: true},
		{line: "END   ; -- comment", result: true},
		{line: "END -- comment", result: false},
		{line: "END -- comment ;", result: false},
		{line: "END \" ; \" -- comment", result: false},
	}

	for _, test := range tests {
		r := endsWithSemicolon(test.line)
		if r != test.result {
			t.Errorf("incorrect semicolon. got %v, want %v", r, test.result)
		}
	}
}

func TestSplitStatements(t *testing.T) {
	t.Parallel()
	// SetVerbose(true)

	type testData struct {
		sql  string
		up   int
		down int
	}

	tt := []testData{
		{sql: multilineSQL, up: 4, down: 1},
		{sql: emptySQL, up: 0, down: 0},
		{sql: emptySQL2, up: 0, down: 0},
		{sql: functxt, up: 2, down: 2},
		{sql: mysqlChangeDelimiter, up: 4, down: 0},
		{sql: copyFromStdin, up: 1, down: 0},
		{sql: plpgsqlSyntax, up: 2, down: 2},
		{sql: plpgsqlSyntaxMixedStatements, up: 2, down: 2},
	}

	for i, test := range tt {
		// up
		stmts, _, err := parseSQLMigration(strings.NewReader(test.sql), true)
		if err != nil {
			t.Error(errors.Wrapf(err, "tt[%v] unexpected error", i))
		}
		if len(stmts) != test.up {
			t.Errorf("tt[%v] incorrect number of up stmts. got %v (%+v), want %v", i, len(stmts), stmts, test.up)
		}

		// down
		stmts, _, err = parseSQLMigration(strings.NewReader(test.sql), false)
		if err != nil {
			t.Error(errors.Wrapf(err, "tt[%v] unexpected error", i))
		}
		if len(stmts) != test.down {
			t.Errorf("tt[%v] incorrect number of down stmts. got %v (%+v), want %v", i, len(stmts), stmts, test.down)
		}
	}
}

func TestUseTransactions(t *testing.T) {
	t.Parallel()

	type testData struct {
		fileName        string
		useTransactions bool
	}

	tests := []testData{
		{fileName: "./examples/sql-migrations/00001_create_users_table.sql", useTransactions: true},
		{fileName: "./examples/sql-migrations/00002_rename_root.sql", useTransactions: true},
		{fileName: "./examples/sql-migrations/00003_no_transaction.sql", useTransactions: false},
	}

	for _, test := range tests {
		f, err := os.Open(test.fileName)
		if err != nil {
			t.Error(err)
		}
		_, useTx, err := parseSQLMigration(f, true)
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
	tt := []string{
		statementBeginNoStatementEnd,
		unfinishedSQL,
		noUpDownAnnotations,
		multiUpDown,
		downFirst,
	}
	for i, sql := range tt {
		_, _, err := parseSQLMigration(strings.NewReader(sql), true)
		if err == nil {
			t.Errorf("expected error on tt[%v] %q", i, sql)
		}
	}
}

var multilineSQL = `-- +goose Up
CREATE TABLE post (
		id int NOT NULL,
		title text,
		body text,
		PRIMARY KEY(id)
);                  -- 1st stmt

-- comment
SELECT 2;           -- 2nd stmt
SELECT 3; SELECT 3; -- 3rd stmt
SELECT 4;           -- 4th stmt

-- +goose Down
-- comment
DROP TABLE post;    -- 1st stmt
`

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

var multiUpDown = `-- +goose Up
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
`

var downFirst = `-- +goose Down
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

var emptySQL = `-- +goose Up
-- This is just a comment`

var emptySQL2 = `

-- comment
-- +goose Up

-- comment
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

var mysqlChangeDelimiter = `
-- +goose Up
-- +goose StatementBegin
DELIMITER | 
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION my_func( str CHAR(255) ) RETURNS CHAR(255) DETERMINISTIC
BEGIN 
  RETURN "Dummy Body"; 
END | 
-- +goose StatementEnd

-- +goose StatementBegin
DELIMITER ; 
-- +goose StatementEnd

select my_func("123") from dual;
-- +goose Down
`

var copyFromStdin = `
-- +goose Up
-- +goose StatementBegin
COPY public.django_content_type (id, app_label, model) FROM stdin;
1	admin	logentry
2	auth	permission
3	auth	group
4	auth	user
5	contenttypes	contenttype
6	sessions	session
\.
-- +goose StatementEnd
`

var plpgsqlSyntax = `
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd
-- +goose StatementBegin
CREATE TRIGGER update_properties_updated_at BEFORE UPDATE ON properties FOR EACH ROW EXECUTE PROCEDURE  update_updated_at_column();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER update_properties_updated_at
-- +goose StatementEnd
-- +goose StatementBegin
DROP FUNCTION update_updated_at_column()
-- +goose StatementEnd
`

var plpgsqlSyntaxMixedStatements = `
-- +goose Up
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';
-- +goose StatementEnd

CREATE TRIGGER update_properties_updated_at
BEFORE UPDATE
ON properties 
FOR EACH ROW EXECUTE PROCEDURE  update_updated_at_column();

-- +goose Down
DROP TRIGGER update_properties_updated_at;
DROP FUNCTION update_updated_at_column();
`
