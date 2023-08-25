package migrationstats

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pressly/goose/v3/internal/sqlparser"
)

type sqlMigration struct {
	useTx              bool
	upCount, downCount int
}

func parseSQLFile(r io.Reader, debug bool) (*sqlMigration, error) {
	by, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	upStatements, txUp, err := sqlparser.ParseSQLMigration(
		bytes.NewReader(by),
		sqlparser.DirectionUp,
		debug,
	)
	if err != nil {
		return nil, err
	}
	downStatements, txDown, err := sqlparser.ParseSQLMigration(
		bytes.NewReader(by),
		sqlparser.DirectionDown,
		debug,
	)
	if err != nil {
		return nil, err
	}
	// This is a sanity check to ensure that the parser is behaving as expected.
	if txUp != txDown {
		return nil, fmt.Errorf("up and down statements must have the same transaction mode")
	}
	return &sqlMigration{
		useTx:     txUp,
		upCount:   len(upStatements),
		downCount: len(downStatements),
	}, nil
}
