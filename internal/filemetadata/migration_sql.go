package filemetadata

import (
	"bytes"
	"fmt"
	"os"

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

func parseSQLFile(filename string) (*sqlMigration, error) {
	by, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	up, txUp, err := sqlparser.ParseSQLMigration(bytes.NewReader(by), sqlparser.DirectionUp, false)
	if err != nil {
		return nil, err
	}
	down, txDown, err := sqlparser.ParseSQLMigration(bytes.NewReader(by), sqlparser.DirectionDown, false)
	if err != nil {
		return nil, err
	}
	if txUp != txDown {
		return nil, fmt.Errorf("parser error: up and down txn mismatch")
	}
	return &sqlMigration{
		useTx:     txUp,
		upCount:   len(up),
		downCount: len(down),
	}, nil
}
