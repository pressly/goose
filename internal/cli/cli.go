package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
)

// Main is the entry point for the CLI.
//
// If an error is returned, it is printed to stderr and the process exits with a non-zero exit code.
// The process is also canceled when an interrupt signal is received. This function and does not
// return.
func Main(opts ...Options) {
	ctx, stop := newContext()
	go func() {
		defer stop()
		if err := Run(ctx, os.Args[1:], opts...); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}()
	// TODO(mf): this looks wonky because we're not waiting for the context to be done. But
	// eventually, I'd like to add a timeout here so we don't hang indefinitely.
	<-ctx.Done()
	os.Exit(0)
}

// Run runs the CLI with the provided arguments. The arguments should not include the command name
// itself, only the arguments to the command, use os.Args[1:].
//
// Options can be used to customize the behavior of the CLI, such as setting the environment,
// redirecting stdout and stderr, and providing a custom filesystem such as embed.FS.
func Run(ctx context.Context, args []string, opts ...Options) error {
	return run(ctx, args, opts...)
}

func newContext() (context.Context, context.CancelFunc) {
	signals := []os.Signal{os.Interrupt}
	if runtime.GOOS != "windows" {
		signals = append(signals, syscall.SIGTERM)
	}
	return signal.NotifyContext(context.Background(), signals...)
}
