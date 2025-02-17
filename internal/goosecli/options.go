package goosecli

import "io"

// Option is a configuration option for the CLI.
type Option interface {
	apply(*config)
}

// WithStdout sets the writer to use for stdout.
func WithStdout(w io.Writer) Option {
	return optionFunc(func(cfg *config) {
		cfg.stdout = w
	})
}

// WithStderr sets the writer to use for stderr.
func WithStderr(w io.Writer) Option {
	return optionFunc(func(cfg *config) {
		cfg.stderr = w
	})
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) { f(cfg) }

type config struct {
	stdout, stderr io.Writer
}
