package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/peterbourgon/ff/v4"
)

const (
	ENV_NO_COLOR = "NO_COLOR"
)

func run(ctx context.Context, args []string, opts ...Options) (retErr error) {
	defer func() {
		if r := recover(); r != nil {
			retErr = fmt.Errorf("panic: %v", r)
		}
	}()
	st, err := newStateWithDefaults(opts...)
	if err != nil {
		return err
	}

	root := newRootCommand(st)
	// Add subcommands
	commands := []func(*state) (*ff.Command, error){
		newStatusCommand,
	}
	for _, cmd := range commands {
		c, err := cmd(st)
		if err != nil {
			return err
		}
		root.Subcommands = append(root.Subcommands, c)
	}

	// Parse the flags and return help if requested.
	if err := root.Parse(
		args,
		ff.WithEnvVarPrefix("GOOSE"), // Support environment variables for all flags
	); err != nil {
		if errors.Is(err, ff.ErrHelp) {
			fmt.Fprintf(st.stderr, "\n%s\n", createHelp(root))
			return nil
		}
		return err
	}
	// TODO(mf): ideally this would be done in the ff package. See open issue:
	// https://github.com/peterbourgon/ff/issues/128
	if err := checkRequiredFlags(root); err != nil {
		return err
	}
	return root.Run(ctx)
}

func newStateWithDefaults(opts ...Options) (*state, error) {
	state := &state{
		environ: os.Environ(),
	}
	for _, opt := range opts {
		if err := opt.apply(state); err != nil {
			return nil, err
		}
	}
	// Set defaults if not set by the caller
	if state.stdout == nil {
		state.stdout = os.Stdout
	}
	if state.stderr == nil {
		state.stderr = os.Stderr
	}
	if state.fsys == nil {
		// Use the default filesystem if not set, reading from the local filesystem.
		state.fsys = func(dir string) (fs.FS, error) { return os.DirFS(dir), nil }
	}
	if state.openConnection == nil {
		// Use the default openConnection function if not set.
		state.openConnection = openConnection
	}
	return state, nil
}

func checkRequiredFlags(cmd *ff.Command) error {
	if cmd != nil {
		cmd = cmd.GetSelected()
	}
	var required []string
	if err := cmd.Flags.WalkFlags(func(f ff.Flag) error {
		name, ok := f.GetLongName()
		if !ok {
			return fmt.Errorf("flag %v doesn't have a long name", f)
		}
		if requiredFlags[name] && !f.IsSet() {
			required = append(required, "--"+name)
		}
		return nil
	}); err != nil {
		return err
	}
	if len(required) > 0 {
		return fmt.Errorf("required flags not set: %v", strings.Join(required, ", "))
	}
	return nil
}

// func coalesce[T comparable](values ...T) (zero T) {
// 	for _, v := range values {
// 		if v != zero {
// 			return v
// 		}
// 	}
// 	return zero
// }
