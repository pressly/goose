package sqlparser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/pressly/goose/v3/internal/check"
)

var (
	debug = false
)

func TestMain(m *testing.M) {
	debug, _ = strconv.ParseBool(os.Getenv("DEBUG_TEST"))
	os.Exit(m.Run())
}

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
		stmts, _, err := ParseSQLMigration(strings.NewReader(test.sql), DirectionUp, debug)
		if err != nil {
			t.Error(fmt.Errorf("tt[%v] unexpected error: %w", i, err))
		}
		if len(stmts) != test.up {
			t.Errorf("tt[%v] incorrect number of up stmts. got %v (%+v), want %v", i, len(stmts), stmts, test.up)
		}

		// down
		stmts, _, err = ParseSQLMigration(strings.NewReader(test.sql), DirectionDown, debug)
		if err != nil {
			t.Error(fmt.Errorf("tt[%v] unexpected error: %w", i, err))
		}
		if len(stmts) != test.down {
			t.Errorf("tt[%v] incorrect number of down stmts. got %v (%+v), want %v", i, len(stmts), stmts, test.down)
		}
	}
}

func TestInvalidUp(t *testing.T) {
	t.Parallel()

	testdataDir := filepath.Join("testdata", "invalid", "up")
	entries, err := os.ReadDir(testdataDir)
	check.NoError(t, err)
	check.NumberNotZero(t, len(entries))

	for _, entry := range entries {
		by, err := os.ReadFile(filepath.Join(testdataDir, entry.Name()))
		check.NoError(t, err)
		_, _, err = ParseSQLMigration(strings.NewReader(string(by)), DirectionUp, false)
		check.HasError(t, err)
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
		_, useTx, err := ParseSQLMigration(f, DirectionUp, debug)
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
		_, _, err := ParseSQLMigration(strings.NewReader(sql), DirectionUp, debug)
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
	// ├── 01.up.golden.sql
	// ├── 02.up.golden.sql
	// ├── 03.up.golden.sql
	// └── input.sql
	tests := []struct {
		Name            string
		StatementsCount int
	}{
		{Name: "test01", StatementsCount: 3},
		{Name: "test02", StatementsCount: 1},
		{Name: "test03", StatementsCount: 1},
		{Name: "test04", StatementsCount: 3},
		{Name: "test05", StatementsCount: 2},
		{Name: "test06", StatementsCount: 5},
		{Name: "test07", StatementsCount: 1},
		{Name: "test08", StatementsCount: 6},
		{Name: "test09", StatementsCount: 1},
	}
	for _, tc := range tests {
		path := filepath.Join("testdata", "valid-up", tc.Name)
		t.Run(tc.Name, func(t *testing.T) {
			testValid(t, path, tc.StatementsCount, DirectionUp)
		})
	}
}

func testValid(t *testing.T, dir string, count int, direction Direction) {
	t.Helper()

	f, err := os.Open(filepath.Join(dir, "input.sql"))
	check.NoError(t, err)
	t.Cleanup(func() { f.Close() })
	statements, _, err := ParseSQLMigration(f, direction, debug)
	check.NoError(t, err)
	check.Number(t, len(statements), count)
	compareStatements(t, dir, statements, direction)
}

func compareStatements(t *testing.T, dir string, statements []string, direction Direction) {
	t.Helper()

	files, err := filepath.Glob(filepath.Join(dir, fmt.Sprintf("*.%s.golden.sql", direction)))
	check.NoError(t, err)
	if len(statements) != len(files) {
		t.Fatalf("mismatch between parsed statements (%d) and golden files (%d), did you check in NN.{up|down}.golden.sql file in %q?", len(statements), len(files), dir)
	}
	for _, goldenFile := range files {
		goldenFile = filepath.Base(goldenFile)
		before, _, ok := strings.Cut(goldenFile, ".")
		if !ok {
			t.Fatal(`failed to cut on file delimiter ".", must be of the format NN.{up|down}.golden.sql`)
		}
		index, err := strconv.Atoi(before)
		check.NoError(t, err)
		index--

		goldenFilePath := filepath.Join(dir, goldenFile)
		by, err := os.ReadFile(goldenFilePath)
		check.NoError(t, err)

		got, want := statements[index], string(by)

		if got != want {
			if isCIEnvironment() {
				t.Errorf("input does not match expected golden file:\n\ngot:\n%s\n\nwant:\n%s\n", got, want)
			} else {
				t.Error("input does not match expected output; diff files with .FAIL to debug")
				t.Logf("\ndiff %v %v",
					filepath.Join("internal", "sqlparser", goldenFilePath+".FAIL"),
					filepath.Join("internal", "sqlparser", goldenFilePath),
				)
				err := os.WriteFile(goldenFilePath+".FAIL", []byte(got), 0644)
				check.NoError(t, err)
			}
		}
	}
}

func isCIEnvironment() bool {
	ok, _ := strconv.ParseBool(os.Getenv("CI"))
	return ok
}

func TestEnvsub(t *testing.T) {
	// Do not run in parallel, as this test sets environment variables.

	// Test valid migrations with ${var} like statements when on are substituted for the whole
	// migration.
	t.Setenv("GOOSE_ENV_REGION", "us_east_")
	t.Setenv("GOOSE_ENV_SET_BUT_EMPTY_VALUE", "")
	t.Setenv("GOOSE_ENV_NAME", "foo")

	tests := []struct {
		Name      string
		DownCount int
		UpCount   int
	}{
		{Name: "test01", UpCount: 4, DownCount: 1},
		{Name: "test02", UpCount: 3, DownCount: 0},
		{Name: "test03", UpCount: 1, DownCount: 0},
	}
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			dir := filepath.Join("testdata", "envsub", tc.Name)
			testValid(t, dir, tc.UpCount, DirectionUp)
			testValid(t, dir, tc.DownCount, DirectionDown)
		})
	}
}

func TestEnvsubError(t *testing.T) {
	t.Parallel()

	s := `
-- +goose ENVSUB ON
-- +goose Up
CREATE TABLE post (
	id int NOT NULL,
	title text,
	${SOME_UNSET_VAR?required env var not set} text,
	PRIMARY KEY(id)
);
`
	_, _, err := ParseSQLMigration(strings.NewReader(s), DirectionUp, debug)
	check.HasError(t, err)
	check.Contains(t, err.Error(), "variable substitution failed: $SOME_UNSET_VAR: required env var not set:")
}

func Test_extractAnnotation(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    annotation
		wantErr func(t *testing.T, err error)
	}{
		{
			name:    "Up",
			input:   "-- +goose Up",
			want:    annotationUp,
			wantErr: check.NoError,
		},
		{
			name:    "Down",
			input:   "-- +goose Down",
			want:    annotationDown,
			wantErr: check.NoError,
		},
		{
			name:    "StmtBegin",
			input:   "-- +goose StatementBegin",
			want:    annotationStatementBegin,
			wantErr: check.NoError,
		},
		{
			name:    "NoTransact",
			input:   "-- +goose NO TRANSACTION",
			want:    annotationNoTransaction,
			wantErr: check.NoError,
		},
		{
			name:    "Unsupported",
			input:   "-- +goose unsupported",
			want:    "",
			wantErr: check.HasError,
		},
		{
			name:    "Empty",
			input:   "-- +goose",
			want:    "",
			wantErr: check.HasError,
		},
		{
			name:    "statement with spaces and Uppercase",
			input:   "-- +goose   UP 	",
			want:    annotationUp,
			wantErr: check.NoError,
		},
		{
			name:    "statement with leading whitespace - error",
			input:   " -- +goose   UP 	",
			want:    "",
			wantErr: check.HasError,
		},
		{
			name:    "statement with leading \t - error",
			input:   "\t-- +goose   UP 	",
			want:    "",
			wantErr: check.HasError,
		},
		{
			name:    "multiple +goose annotations - error",
			input:   "-- +goose +goose Up",
			want:    "",
			wantErr: check.HasError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractAnnotation(tt.input)
			tt.wantErr(t, err)

			check.Equal(t, got, tt.want)
		})
	}
}
