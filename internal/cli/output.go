package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v4"
)

type migrationResultsOutput struct {
	Migrations    []migrationResult `json:"migrations"`
	TotalDuration int64             `json:"total_duration_ms"`
}

type migrationResult struct {
	Type      string `json:"migration_type"`
	Version   int64  `json:"version"`
	Filename  string `json:"filename"`
	Duration  int64  `json:"duration_ms"`
	Direction string `json:"direction"`
	Empty     bool   `json:"empty"`
}

func convertMigrationResult(
	results []*goose.MigrationResult,
	totalDuration time.Duration,
) migrationResultsOutput {
	output := migrationResultsOutput{
		Migrations:    make([]migrationResult, 0, len(results)),
		TotalDuration: totalDuration.Milliseconds(),
	}
	for _, result := range results {
		output.Migrations = append(output.Migrations, migrationResult{
			Type:      string(result.Migration.Type),
			Version:   result.Migration.Version,
			Filename:  filepath.Base(result.Migration.Source),
			Duration:  result.Duration.Milliseconds(),
			Direction: result.Direction,
			Empty:     result.Empty,
		})
	}
	return output
}

func printMigrationResult(
	results []*goose.MigrationResult,
	totalDuration time.Duration,
	useJson bool,
) error {
	if len(results) == 0 {
		fmt.Println("no migrations to run")
		return nil
	}
	if useJson {
		data := convertMigrationResult(results, totalDuration)
		return json.NewEncoder(os.Stdout).Encode(data)
	}
	for _, r := range results {
		if !r.Empty {
			fmt.Printf("OK   %s (%s)\n", filepath.Base(r.Migration.Source), truncateDuration(r.Duration))
		} else {
			fmt.Printf("EMPTY %s (%s)\n", filepath.Base(r.Migration.Source), truncateDuration(r.Duration))
		}
	}
	return nil
}

func truncateDuration(d time.Duration) time.Duration {
	for _, v := range []time.Duration{
		time.Second,
		time.Millisecond,
		time.Microsecond,
	} {
		if d > v {
			return d.Round(v / time.Duration(100))
		}
	}
	return d
}
