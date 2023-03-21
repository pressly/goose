package migration

import (
	"bytes"
	"io"
	"io/fs"

	"github.com/pressly/goose/v4/internal/sqlparser"
)

// parseSQLMigrations parses all SQL migrations in BOTH direction. If a migration has already been
// parsed, it will not be parsed again.
//
// Note, this function will mutate the migrations.
func ParseSQL(fsys fs.FS, debug bool, migrations []*Migration) error {
	for _, m := range migrations {
		if m.IsSQL() && !m.SQLParsed {
			parsedSQLMigration, err := parseSQL(fsys, m.Fullpath, debug, sqlparser.DirectionAll)
			if err != nil {
				return err
			}
			m.SQLParsed = true
			m.SQL = parsedSQLMigration
		}
	}
	return nil
}

func parseSQL(fsys fs.FS, filename string, debug bool, d sqlparser.Direction) (*SQL, error) {
	r, err := fsys.Open(filename)
	if err != nil {
		return nil, err
	}
	by, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	if err := r.Close(); err != nil {
		return nil, err
	}
	m := new(SQL)
	if d == sqlparser.DirectionAll || d == sqlparser.DirectionUp {
		m.UpStatements, m.UseTx, err = sqlparser.ParseSQLMigration(
			bytes.NewReader(by),
			sqlparser.DirectionUp,
			debug,
		)
		if err != nil {
			return nil, err
		}
	}
	if d == sqlparser.DirectionAll || d == sqlparser.DirectionDown {
		m.DownStatements, m.UseTx, err = sqlparser.ParseSQLMigration(
			bytes.NewReader(by),
			sqlparser.DirectionDown,
			debug,
		)
		if err != nil {
			return nil, err
		}
	}
	return m, nil
}
