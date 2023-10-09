package migrate

import (
	"bytes"
	"io"
	"io/fs"

	"github.com/pressly/goose/v3/internal/sqlparser"
)

// ParseSQL parses all SQL migrations in BOTH directions. If a migration has already been parsed, it
// will not be parsed again.
//
// Important: This function will mutate SQL migrations.
func ParseSQL(fsys fs.FS, debug bool, migrations []*Migration) error {
	for _, m := range migrations {
		if m.Type == TypeSQL && !m.SQLParsed {
			parsedSQLMigration, err := parseSQL(fsys, m.Fullpath, parseAll, debug)
			if err != nil {
				return err
			}
			m.SQLParsed = true
			m.SQL = parsedSQLMigration
		}
	}
	return nil
}

// parse is used to determine which direction to parse the SQL migration.
type parse int

const (
	// parseAll parses all SQL statements in BOTH directions.
	parseAll parse = iota + 1
	// parseUp parses all SQL statements in the UP direction.
	parseUp
	// parseDown parses all SQL statements in the DOWN direction.
	parseDown
)

func parseSQL(fsys fs.FS, filename string, p parse, debug bool) (*SQL, error) {
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
	s := new(SQL)
	if p == parseAll || p == parseUp {
		s.UpStatements, s.UseTx, err = sqlparser.ParseSQLMigration(
			bytes.NewReader(by),
			sqlparser.DirectionUp,
			debug,
		)
		if err != nil {
			return nil, err
		}
	}
	if p == parseAll || p == parseDown {
		s.DownStatements, s.UseTx, err = sqlparser.ParseSQLMigration(
			bytes.NewReader(by),
			sqlparser.DirectionDown,
			debug,
		)
		if err != nil {
			return nil, err
		}
	}
	return s, nil
}
