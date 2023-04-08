package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

// Run is the entry point for the goose CLI. It parses the command line
// arguments and executes the appropriate command.
//
// The supplied args should not include the name of the executable.
func Run(args []string) error {
	rootCmd, rootConfig := newRootCmd()

	rootCmd.Subcommands = []*ffcli.Command{
		// Up commands.
		newUpCmd(rootConfig),
		newUpToCmd(rootConfig),
		newUpByOneCmd(rootConfig),
		// Down commands.
		newDownToCmd(rootConfig),
		newDownCmd(rootConfig),

		newRedoCmd(rootConfig),
		newStatusCmd(rootConfig),
		newVersionCmd(rootConfig),
		newFixCmd(rootConfig),
		newCreateCmd(rootConfig),
		newEnvCmd(rootConfig),
	}

	// for _, c := range rootCmd.Subcommands {
	// 	if c.UsageFunc == nil {
	// 		c.UsageFunc = func(c *ffcli.Command) string {
	// 			return "This overrides the default usage function."
	// 		}
	// 	}
	// }

	if err := rootCmd.Parse(args); err != nil {
		return fmt.Errorf("parsing error: %w", err)
	}

	// This is where we can validate the root config after parsing the command line flags.
	_ = rootConfig

	// Here we can add additional commands but not advertised in the help text.

	return rootCmd.Run(context.Background())
}

type stringSet []string

func (ss *stringSet) Set(value string) error {
	for _, existing := range *ss {
		if value == existing {
			return errors.New("duplicate")
		}
	}
	(*ss) = append(*ss, value)
	return nil
}

func (ss *stringSet) String() string {
	return strings.Join(*ss, ", ")
}

func printMigrationResult(
	results []*goose.MigrationResult,
	totalDuration time.Duration,
	useJson bool,
) error {
	if useJson {
		data := convertMigrationResult(results, totalDuration)
		return json.NewEncoder(os.Stdout).Encode(data)
	}
	// TODO: print a table
	for _, result := range results {
		fmt.Println(result)
	}
	return nil
}

type migrationsOutput struct {
	Migrations    []migrationResultOutput `json:"migrations"`
	TotalDuration int64                   `json:"total_duration_ms"`
}

type migrationResultOutput struct {
	Type      string `json:"migration_type"`
	Version   int64  `json:"version"`
	Filename  string `json:"filename"`
	Duration  int64  `json:"duration_ms"`
	Direction string `json:"direction"`
}

func convertMigrationResult(
	results []*goose.MigrationResult,
	totalDuration time.Duration,
) migrationsOutput {
	output := migrationsOutput{
		Migrations:    make([]migrationResultOutput, 0, len(results)),
		TotalDuration: totalDuration.Milliseconds(),
	}
	for _, result := range results {
		output.Migrations = append(output.Migrations, migrationResultOutput{
			Type:      string(result.Migration.Type),
			Version:   result.Migration.Version,
			Filename:  filepath.Base(result.Migration.Source),
			Duration:  result.Duration.Milliseconds(),
			Direction: result.Direction,
		})
	}
	return output
}

// func truncateDuration(d time.Duration) time.Duration {
// 	for _, v := range []time.Duration{
// 		time.Second,
// 		time.Millisecond,
// 		time.Microsecond,
// 	} {
// 		if d > v {
// 			return d.Round(v / time.Duration(100))
// 		}
// 	}
// 	return d
// }
