package goosecli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/mfridman/cli"
)

// Main is the entry point for the goose CLI.
//
// If an error is returned, it is printed to stderr and the process exits with a non-zero exit code.
// The process is also canceled when an interrupt signal is received. This function and does not
// return.
func Main(opts ...Option) {
	ctx, stop := newContext()

	go func() {
		defer stop()

		if err := Run(ctx, os.Args[1:], opts...); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	os.Exit(0)
}

// Run the CLI with the provided arguments. The arguments should not include the command name
// itself, only the arguments to the command, use os.Args[1:].
//
// Options can be used to customize the behavior of the CLI, such as setting the environment,
// redirecting stdout and stderr, and providing a custom filesystem such as embed.FS.
func Run(ctx context.Context, args []string, opts ...Option) error {
	var cfg config
	for _, opt := range opts {
		opt.apply(&cfg)
	}
	return run(ctx, args, cfg)
}

func newContext() (context.Context, context.CancelFunc) {
	signals := []os.Signal{os.Interrupt}
	if runtime.GOOS != "windows" {
		signals = append(signals, syscall.SIGTERM)
	}
	return signal.NotifyContext(context.Background(), signals...)
}

/*

 	up                   Migrate the DB to the most recent version available
    up-by-one            Migrate the DB up by 1
    up-to VERSION        Migrate the DB to a specific VERSION

    down                 Roll back the version by 1
    down-to VERSION      Roll back to a specific VERSION

    ??? redo                 Re-run the latest migration
    ??? reset                Roll back all migrations

	status               Dump the migration status for the current DB
    version              Print the current version of the database
    create NAME [sql|go] Creates new migration file with the current timestamp
    fix                  Apply sequential ordering to migrations
    validate             Check migration files without running them

*/

func run(ctx context.Context, args []string, cfg config) error {
	commands := []*cli.Command{
		up,
		upByOne,
		upTo,
		down,
		downTo,
		status,
		version,
		create,
		fix,
		validate,
	}
	// Add all subcommands to the root command.
	root.SubCommands = append(root.SubCommands, commands...)

	if err := cli.Parse(root, args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			fmt.Fprintf(cfg.stdout, "%s\n", cli.DefaultUsage(root))
			return nil
		}
		return fmt.Errorf("error: %w", err)
	}

	options := &cli.RunOptions{
		Stdout: cfg.stdout,
		Stderr: cfg.stderr,
	}
	if err := cli.Run(ctx, root, options); err != nil {
		return fmt.Errorf("error: %w", err)
	}
	return nil
}
