package migrationstats

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/pressly/goose/v3"
)

// FileWalker walks all files for GatherStats.
type FileWalker interface {
	// Walk invokes fn for each file.
	Walk(fn func(filename string, r io.Reader) error) error
}

// Stats contains the stats for a migration file.
type Stats struct {
	// FileName is the name of the file.
	FileName string
	// Version is the version of the migration.
	Version int64
	// Tx is true if the .sql migration file has a +goose NO TRANSACTION annotation
	// or the .go migration file calls AddMigrationNoTx.
	Tx bool
	// UpCount is the number of statements in the Up migration.
	UpCount int
	// DownCount is the number of statements in the Down migration.
	DownCount int
}

// GatherStats returns the migration file stats.
func GatherStats(fw FileWalker, debug bool) ([]*Stats, error) {
	var stats []*Stats
	err := fw.Walk(func(filename string, r io.Reader) error {
		version, err := goose.NumericComponent(filename)
		if err != nil {
			return fmt.Errorf("failed to get version from file %q: %w", filename, err)
		}
		var up, down int
		var tx bool
		switch filepath.Ext(filename) {
		case ".sql":
			m, err := parseSQLFile(r, debug)
			if err != nil {
				return fmt.Errorf("failed to parse file %q: %w", filename, err)
			}
			up, down = m.upCount, m.downCount
			tx = m.useTx
		case ".go":
			m, err := parseGoFile(r)
			if err != nil {
				return fmt.Errorf("failed to parse file %q: %w", filename, err)
			}
			up, down = nilAsNumber(m.upFuncName), nilAsNumber(m.downFuncName)
			tx = *m.useTx
		}
		stats = append(stats, &Stats{
			FileName:  filename,
			Version:   version,
			Tx:        tx,
			UpCount:   up,
			DownCount: down,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func nilAsNumber(s string) int {
	if s != "nil" {
		return 1
	}
	return 0
}
