package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

var commands = map[string]func(*rootConfig) *ffcli.Command{
	"create":    newCreateCmd,
	"down-to":   newDownToCmd,
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

// Run is the entry point for the goose CLI. It parses the command line arguments and executes the
// appropriate command.
//
// The supplied args should not include the name of the executable.
func Run(args []string) error {
	rootCmd, rootConfig := newRootCmd(os.Stdout)
	// Add subcommands. These will be advertised in the help text.
	for _, cmd := range commands {
		subcommand := cmd(rootConfig)
		subcommand.Options = []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		}
		rootCmd.Subcommands = append(rootCmd.Subcommands, subcommand)
	}
	// Set the usage function for all subcommands, if not already set.
	// for _, c := range rootCmd.Subcommands {
	// 	if c.UsageFunc == nil {
	// 		c.UsageFunc = usageFunc
	// 	}
	// }
	// Parse the command line flags.
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

// func truncateDuration(d time.Duration) time.Duration {
//  for _, v := range []time.Duration{
//      time.Second,
//      time.Millisecond,
//      time.Microsecond,
//  } {
//      if d > v {
//          return d.Round(v / time.Duration(100))
//      }
//  }
//  return d
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
