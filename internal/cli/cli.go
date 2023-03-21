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
	"validate":  newValidateCmd,
	"version":   newVersionCmd,
}

var (
	errNoDir      = errors.New("--dir or GOOSE_DIR must be set")
	errNoDBString = errors.New("--dbstring or GOOSE_DBSTRING must be set")
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
		return nil, errNoDBString
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
		var lockMode goose.LockMode
		switch pf.lockMode {
		case "none":
			lockMode = goose.LockModeNone
		case "advisory-session":
			lockMode = goose.LockModeAdvisorySession
		default:
			return nil, fmt.Errorf("invalid lock mode: %q", pf.lockMode)
		}
		opt = opt.SetLockMode(lockMode)
	}
	return goose.NewProvider(gooseDialect, db, opt)
}
