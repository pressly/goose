package filemetadata

import (
	"fmt"
	"io"

	"github.com/pressly/goose/v3/internal/sqlparser"
)

type sqlMigration struct {
	useTx              bool
	upCount, downCount int
}

func convertSQLMigration(s *sqlMigration) *FileMetadata {
	return &FileMetadata{
		FileType:  "sql",
		Tx:        s.useTx,
		UpCount:   s.upCount,
		DownCount: s.downCount,
	}
}

func parseSQLFile(r io.Reader, debug bool) (*sqlMigration, error) {
	upStatements, txUp, err := sqlparser.ParseSQLMigration(r, sqlparser.DirectionUp, debug)
	if err != nil {
		return nil, err
	}
	downStatements, txDown, err := sqlparser.ParseSQLMigration(r, sqlparser.DirectionDown, debug)
	if err != nil {
		return nil, err
	}
	// This case should never happen. Within a single .sql file if a +goose NO TRANSACTION
	// annotation is set it must apply to the entire file, which includes all up
	// and down statements.
	if txUp != txDown {
		return nil, fmt.Errorf("up and down txn do not match")
	}
	return &sqlMigration{
		useTx:     txUp,
		upCount:   len(upStatements),
		downCount: len(downStatements),
	}, nil
}
