package sqlparser

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/pressly/goose/v3/internal/check"
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
		stmts, _, err := ParseSQLMigration(strings.NewReader(test.sql), true)
		if err != nil {
			t.Error(fmt.Errorf("tt[%v] unexpected error: %w", i, err))
		}
		if len(stmts) != test.up {
			t.Errorf("tt[%v] incorrect number of up stmts. got %v (%+v), want %v", i, len(stmts), stmts, test.up)
		}

		// down
		stmts, _, err = ParseSQLMigration(strings.NewReader(test.sql), false)
		if err != nil {
			t.Error(fmt.Errorf("tt[%v] unexpected error: %w", i, err))
		}
		if len(stmts) != test.down {
			t.Errorf("tt[%v] incorrect number of down stmts. got %v (%+v), want %v", i, len(stmts), stmts, test.down)
		}
	}
}

func TestKeepEmptyLines(t *testing.T) {
	stmts, _, err := ParseSQLMigration(strings.NewReader(emptyLineSQL), true)
	if err != nil {
		t.Errorf("Failed to parse SQL migration. %v", err)
	}
	expected := `INSERT INTO post (id, title, body)
VALUES ('id_01', 'my_title', '
this is an insert statement including empty lines.

empty (blank) lines can be meaningful.

leave the lines to keep the text syntax.
');
`
	if stmts[0] != expected {
		t.Errorf("incorrect stmts. got %v, want %v", stmts, expected)
	}
}

func TestUseTransactions(t *testing.T) {
	t.Parallel()

	type testData struct {
		fileName        string
		useTransactions bool
	}

	tests := []testData{
		{fileName: "testdata/valid-txn/00001_create_users_table.sql", useTransactions: true},
		{fileName: "testdata/valid-txn/00002_rename_root.sql", useTransactions: true},
		{fileName: "testdata/valid-txn/00003_no_transaction.sql", useTransactions: false},
	}

	for _, test := range tests {
		f, err := os.Open(test.fileName)
		if err != nil {
			t.Error(err)
		}
		_, useTx, err := ParseSQLMigration(f, true)
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
		_, _, err := ParseSQLMigration(strings.NewReader(sql), true)
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

var emptyLineSQL = `-- +goose Up
INSERT INTO post (id, title, body)
VALUES ('id_01', 'my_title', '
this is an insert statement including empty lines.

empty (blank) lines can be meaningful.

leave the lines to keep the text syntax.
');

-- +goose Down
TRUNCATE TABLE post; 
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

func TestValidUp(t *testing.T) {
	t.Parallel()
	// Test valid "up" parser logic.
	//
	// This test expects each directory, such as: internal/sqlparser/testdata/valid-up/test01
	//
	// to contain exactly one migration file called "input.sql". We read this file and pass it
	// to the parser. Then we compare the statements against the golden files.
	// Each golden file is equivalent to one statement.
	//
	// ├── 01.golden.sql
	// ├── 02.golden.sql
	// ├── 03.golden.sql
	// └── input.sql
	tests := []struct {
		Name            string
		StatementsCount int
	}{
		{Name: "test01", StatementsCount: 3},
		{Name: "test02", StatementsCount: 1},
		{Name: "test03", StatementsCount: 1},
		{Name: "test04", StatementsCount: 2},
		{Name: "test05", StatementsCount: 2},
		{Name: "test06", StatementsCount: 3},
	}
	for _, tc := range tests {
		path := filepath.Join("testdata", "valid-up", tc.Name)
		t.Run(tc.Name, func(t *testing.T) {
			testValidUp(t, path, tc.StatementsCount)
		})
	}
}

func testValidUp(t *testing.T, dir string, count int) {
	t.Helper()

	f, err := os.Open(filepath.Join(dir, "input.sql"))
	check.NoError(t, err)
	t.Cleanup(func() { f.Close() })
	statements, _, err := ParseSQLMigration(f, true)
	check.NoError(t, err)
	check.Number(t, len(statements), count)
	compareStatements(t, dir, statements)
}

func compareStatements(t *testing.T, dir string, statements []string) {
	t.Helper()

	files, err := os.ReadDir(dir)
	check.NoError(t, err)
	for _, goldenFile := range files {
		if goldenFile.Name() == "input.sql" {
			continue
		}
		if !strings.HasSuffix(goldenFile.Name(), ".golden.sql") {
			t.Fatalf("expecting golden file with format <name>.golden.sql: got: %q. Try running `make clean` to remove previous failed files?", goldenFile.Name())
		}
		before, _, ok := cut(goldenFile.Name(), ".")
		if !ok {
			t.Fatal(`failed to cut on file delimiter ".", must be of the format NN.golden.sql`)
		}
		index, err := strconv.Atoi(before)
		check.NoError(t, err)
		index--

		goldenFilePath := filepath.Join(dir, goldenFile.Name())
		by, err := os.ReadFile(goldenFilePath)
		check.NoError(t, err)

		got, want := strings.TrimSpace(statements[index]), strings.TrimSpace(string(by))

		if got != want {
			if isCIEnvironment() {
				t.Errorf("input does not match expected golden file:\n\ngot:\n%s\n\nwant:\n%s\n", got, want)
			} else {
				t.Error("input does not match expected output; diff files with .FAIL to debug")
				t.Logf("\ndiff %v %v",
					filepath.Join("internal", "sqlparser", goldenFilePath+".FAIL"),
					filepath.Join("internal", "sqlparser", goldenFilePath),
				)
				err := ioutil.WriteFile(goldenFilePath+".FAIL", []byte(got+"\n"), 0644)
				check.NoError(t, err)
			}
		}
	}
}

// copied directly from strings.Cut (go1.18) to support older Go versions.
// In the future, replace this with the upstream function.
func cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}

func isCIEnvironment() bool {
	ok, _ := strconv.ParseBool(os.Getenv("CI"))
	return ok
}
