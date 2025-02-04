package goosecli

import "io"

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

type Option interface {
	apply(*config)
}

type optionFunc func(*config)

func (f optionFunc) apply(cfg *config) { f(cfg) }

type config struct {
	stdout, stderr io.Writer
}
