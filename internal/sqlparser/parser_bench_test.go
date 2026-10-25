package sqlparser

import (
	"strings"
	"testing"
	"testing/fstest"
)

// benchSmallMigration is a typical migration file of a few statements.
const benchSmallMigration = `-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id serial PRIMARY KEY,
    username text NOT NULL UNIQUE,
    name text NOT NULL,
    surname text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);
-- +goose StatementEnd
CREATE INDEX idx_users_name ON users (name);
INSERT INTO users (username, name, surname) VALUES ('a', 'b', 'c');
ALTER TABLE users ADD COLUMN email text; -- trailing comment

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
DROP FUNCTION set_updated_at();
DROP INDEX idx_users_name;
DROP TABLE users;
`

// benchLargeMigration has 1000 up and 1000 down statements.
var benchLargeMigration = func() string {
	var sb strings.Builder
	sb.WriteString("-- +goose Up\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("INSERT INTO t (a, b, c) VALUES (1, 'hello world', 'xyz'); -- comment\n")
	}
	sb.WriteString("-- +goose Down\n")
	for i := 0; i < 1000; i++ {
		sb.WriteString("DELETE FROM t WHERE a = 1;\n")
	}
	return sb.String()
}()

// benchLongLineMigration has a single 1 MiB statement on one line.
var benchLongLineMigration = "-- +goose Up\nINSERT INTO t (v) VALUES ('" +
	strings.Repeat("x", 1024*1024) + "');\n-- +goose Down\nDELETE FROM t;\n"

func BenchmarkParseSQLMigration(b *testing.B) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{"small", benchSmallMigration},
		{"large", benchLargeMigration},
		{"long_line", benchLongLineMigration},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(tc.input)))
			for i := 0; i < b.N; i++ {
				if _, _, err := ParseSQLMigration(strings.NewReader(tc.input), DirectionUp, false); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkParseAllFromFS(b *testing.B) {
	fsys := fstest.MapFS{
		"00001_small.sql": &fstest.MapFile{Data: []byte(benchSmallMigration)},
		"00002_large.sql": &fstest.MapFile{Data: []byte(benchLargeMigration)},
	}
	for _, name := range []string{"00001_small.sql", "00002_large.sql"} {
		b.Run(strings.TrimSuffix(strings.TrimPrefix(name, "0000"), ".sql"), func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := ParseAllFromFS(fsys, name, false); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkEndsWithSemicolon(b *testing.B) {
	lines := []string{
		"CREATE TABLE t (id int);",
		"    id serial PRIMARY KEY,",
		"ALTER TABLE users ADD COLUMN email text; -- trailing comment",
		"END \" ; \" -- comment",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, line := range lines {
			_ = endsWithSemicolon(line)
		}
	}
}
