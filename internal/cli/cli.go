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

var commands = map[string]func(*rootConfig) *ffcli.Command{
	"create":    newCreateCmd,
	"down":      newDownCmd,
	"env":       newEnvCmd,
	"fix":       newFixCmd,
	"redo":      newRedoCmd,
	"status":    newStatusCmd,
	"up-by-one": newUpByOneCmd,
	"up-to":     newUpToCmd,
	"up":        newUpCmd,
	"version":   newVersionCmd,
}

// Run is the entry point for the goose CLI. It parses the command line
// arguments and executes the appropriate command.
//
// The supplied args should not include the name of the executable.
func Run(args []string) error {
	rootCmd, rootConfig := newRootCmd(os.Stdout)
	// Add subcommands. These will be advertised in the help text.
	for _, cmd := range commands {
		rootCmd.Subcommands = append(rootCmd.Subcommands, cmd(rootConfig))
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

func newGooseProvider(root *rootConfig) (*goose.Provider, error) {
	db, gooseDialect, err := openConnection(root.dbstring)
	if err != nil {
		return nil, err
	}
	opt := goose.DefaultOptions().
		SetVerbose(root.verbose).
		SetNoVersioning(root.noVersioning).
		SetAllowMissing(root.allowMissing)

	if len(root.excludeFilenames) > 0 {
		opt = opt.SetExcludeFilenames(root.excludeFilenames...)
	}
	if root.dir != "" {
		opt = opt.SetDir(root.dir)
	}
	if root.table != "" {
		opt = opt.SetTableName(root.table)
	}
	if root.lockMode != "" {
		var lockMode goose.LockMode
		switch root.lockMode {
		case "none":
			lockMode = goose.LockModeNone
		case "advisory-session":
			lockMode = goose.LockModeAdvisorySession
		default:
			return nil, fmt.Errorf("invalid lock mode: %s", root.lockMode)
		}
		opt = opt.SetLockMode(lockMode)
	}

	return goose.NewProvider(gooseDialect, db, opt)
}
