package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

var commands = map[string]func(*rootConfig) *ffcli.Command{
	"create":    newCreateCmd,
	"down-to":   newDownToCmd,
	"down":      newDownCmd,
	"env":       newEnvCmd,
	"fix":       newFixCmd,
	"init":      newInitCmd,
	"redo":      newRedoCmd,
	"status":    newStatusCmd,
	"up-by-one": newUpByOneCmd,
	"up-to":     newUpToCmd,
	"up":        newUpCmd,
	"version":   newVersionCmd,
}

var (
	errNoDir = errors.New("--dir or GOOSE_DIR must be set")
)

// Run is the entry point for the goose CLI. It parses the command line arguments and executes the
// appropriate command.
//
// The supplied args should not include the name of the executable.
func Run(args []string) error {
	rootCmd, rootConfig := newRootCmd()
	// Add subcommands. These will be advertised in the help text.
	for _, cmd := range commands {
		rootCmd.Subcommands = append(rootCmd.Subcommands, cmd(rootConfig))
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

func parseLockMode(s string) (goose.LockMode, error) {
	switch s {
	case "none":
		return goose.LockModeNone, nil
	case "advisory-session":
		return goose.LockModeAdvisorySession, nil
	default:
		return 0, fmt.Errorf("invalid lock mode: %s", s)
	}
}

func coalesce[T comparable](values ...T) (zero T) {
	for _, v := range values {
		if v != zero {
			return v
		}
	}
	return zero
}

func newGooseProvider(root *rootConfig, pf *providerFlags) (*goose.Provider, error) {
	dir := coalesce(pf.dir, GOOSE_DIR)
	if dir == "" {
		return nil, errNoDir
	}
	dbstring := coalesce(pf.dbstring, GOOSE_DBSTRING)
	if dbstring == "" {
		return nil, fmt.Errorf("--dbstring or GOOSE_DBSTRING must be set")
	}

	db, gooseDialect, err := openConnection(dbstring)
	if err != nil {
		return nil, err
	}
	tableName := coalesce(pf.table, GOOSE_TABLE)

	opt := goose.DefaultOptions().
		SetDir(dir).
		SetTableName(tableName).
		SetVerbose(root.verbose).
		SetAllowMissing(pf.allowMissing).
		SetNoVersioning(pf.noVersioning).
		SetExcludeFilenames(pf.excludeFilenames...)

	if pf.lockMode != "" {
		lockMode, err := parseLockMode(pf.lockMode)
		if err != nil {
			return nil, err
		}
		opt = opt.SetLockMode(lockMode)
	}
	return goose.NewProvider(gooseDialect, db, opt)
}
