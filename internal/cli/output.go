package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pressly/goose/v4"
	"go.uber.org/multierr"
)

type resultsOutput struct {
	MigrationResults []result `json:"migrations"`
	TotalDuration    int64    `json:"total_duration_ms"`
	HasError         bool     `json:"has_error"`
}

type result struct {
	Type      string `json:"migration_type"`
	Version   int64  `json:"version"`
	Filename  string `json:"filename"`
	Duration  int64  `json:"duration_ms"`
	Direction string `json:"direction"`
	Empty     bool   `json:"empty"`
	Error     string `json:"error,omitempty"`
}

func printResult(
	migrationResults []*goose.MigrationResult,
	err error,
	totalDuration time.Duration,
	useJSON bool,
) error {
	output := resultsOutput{
		MigrationResults: convertResult(migrationResults),
		TotalDuration:    totalDuration.Milliseconds(),
		HasError:         err != nil,
	}
	if useJSON {
		encodeErr := json.NewEncoder(os.Stdout).Encode(output)
		return multierr.Append(err, encodeErr)
	}
	if len(migrationResults) == 0 {
		fmt.Fprintln(os.Stdout, "no migrations to run")
		return err
	}
	for _, r := range migrationResults {
		if !r.Empty {
			fmt.Fprintf(os.Stdout, "OK   %s (%s)\n", filepath.Base(r.Source), truncateDuration(r.Duration))
		} else {
			fmt.Fprintf(os.Stdout, "EMPTY %s (%s)\n", filepath.Base(r.Source), truncateDuration(r.Duration))
		}
	}
	if err == nil {
		fmt.Fprintf(os.Stdout, "\nsuccessfully applied %d migrations in %v", len(migrationResults), truncateDuration(totalDuration))
	} else {
		fmt.Fprintf(os.Stderr, "\npartial migration error: %v", err)
	}
	return err
}

func convertResult(results []*goose.MigrationResult) []result {
	output := make([]result, 0, len(results))
	for _, r := range results {
		result := result{
			Type:      strings.ToLower(r.Type.String()),
			Version:   r.Version,
			Filename:  filepath.Base(r.Source),
			Duration:  r.Duration.Milliseconds(),
			Direction: r.Direction,
			Empty:     r.Empty,
		}
		if r.Error != nil {
			result.Error = r.Error.Error()
		}
		output = append(output, result)
	}
	return output
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
