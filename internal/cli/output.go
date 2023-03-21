package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/pressly/goose/v4"
)

// TODO(mf): add JSON output

//lint:ignore U1000 Ignore unused code for now
type resultsOutput struct {
	MigrationResults []result `json:"migrations"`
	TotalDuration    int64    `json:"total_duration_ms"`
	HasError         bool     `json:"has_error"`
	Error            string   `json:"error,omitempty"`
}

//lint:ignore U1000 Ignore unused code for now
type result struct {
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
	switch {
	case err == nil, errors.Is(err, goose.ErrNoMigration):
		printMigrations(os.Stdout, migrationResults)
		if count := len(migrationResults); count > 0 {
			msg := "migration"
			if count > 1 {
				msg += "s"
			}
			fmt.Printf("applied %d %s in %v\n", count, msg, truncateDuration(totalDuration))
		} else {
			fmt.Println("no migrations to run")
		}
		return nil
	}
	if perr := new(goose.PartialError); errors.As(err, &perr) {
		printMigrations(os.Stdout, perr.Results)
	}
	return err
}

func printMigrations(w io.Writer, results []*goose.MigrationResult) {
	tw := tabwriter.NewWriter(w, 0, 2, 6, ' ', 0)
	for i, r := range results {
		msg := "OK"
		if r.Empty {
			msg = "EMPTY"
		}
		fmt.Fprintf(tw, "%s\t%s (%s)\n", msg, filepath.Base(r.Fullpath), truncateDuration(r.Duration))
		if i == len(results)-1 {
			fmt.Fprintln(tw)
		}
	}
	tw.Flush()
}

//lint:ignore U1000 Ignore unused code for now
func convertResult(results []*goose.MigrationResult) []result {
	output := make([]result, 0, len(results))
	for _, r := range results {
		result := result{
			Version:   r.Version,
			Filename:  filepath.Base(r.Fullpath),
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
