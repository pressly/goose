package cli

import (
	"database/sql"
	"fmt"
	"io"
	"io/fs"

	"github.com/pressly/goose/v3"
)

// Options are used to configure the command execution and are passed to the Run or Main function.
type Options interface {
	apply(*state) error
}

type optionFunc func(*state) error

func (f optionFunc) apply(s *state) error { return f(s) }

// WithEnviron sets the environment variables for the command. This will overwrite the current
// environment, primarily useful for testing.
func WithEnviron(env []string) Options {
	return optionFunc(func(s *state) error {
		s.environ = env
		return nil
	})
}

// WithStdout sets the writer for stdout.
func WithStdout(w io.Writer) Options {
	return optionFunc(func(s *state) error {
		if w == nil {
			return fmt.Errorf("stdout cannot be nil")
		}
		if s.stdout != nil {
			return fmt.Errorf("stdout already set")
		}
		s.stdout = w
		return nil
	})
}

// WithStderr sets the writer for stderr.
func WithStderr(w io.Writer) Options {
	return optionFunc(func(s *state) error {
		if w == nil {
			return fmt.Errorf("stderr cannot be nil")
		}
		if s.stderr != nil {
			return fmt.Errorf("stderr already set")
		}
		s.stderr = w
		return nil
	})
}

// WithFilesystem takes a function that returns a filesystem for the given directory. The directory
// will be the value of the --dir flag passed to the command. A typical use case is to use
// [embed.FS] or [fstest.MapFS]. For example:
//
//	fsys := fstest.MapFS{
//	    "migrations/001_foo.sql": {Data: []byte(`-- +goose Up`)},
//	}
//	err := cli.Run(context.Background(), os.Args[1:], cli.WithFilesystem(fsys.Sub))
//
// The above example will run the command with the filesystem provided by [fsys.Sub].
func WithFilesystem(fsys func(dir string) (fs.FS, error)) Options {
	return optionFunc(func(s *state) error {
		if fsys == nil {
			return fmt.Errorf("filesystem cannot be nil")
		}
		if s.fsys != nil {
			return fmt.Errorf("filesystem already set")
		}
		s.fsys = fsys
		return nil
	})
}

// WithOpenConnection sets the function that opens a connection to the database from a DSN string.
// The function should return the dialect and the database connection. The dbstring will typically
// be a DSN, such as "postgres://user:password@localhost/dbname" or "sqlite3://file.db" and it is up
// to the function to parse it.
func WithOpenConnection(open func(dbstring string) (*sql.DB, goose.Dialect, error)) Options {
	return optionFunc(func(s *state) error {
		if open == nil {
			return fmt.Errorf("open connection function cannot be nil")
		}
		if s.openConnection != nil {
			return fmt.Errorf("open connection function already set")
		}
		s.openConnection = open
		return nil
	})
}

// WithVersion sets the version string for the command. This is typically set by the build system
// when the binary is built. It is used to print the version when the --version flag is passed.
func WithVersion(version string) Options {
	return optionFunc(func(s *state) error {
		if version == "" {
			return fmt.Errorf("version cannot be empty")
		}
		if s.version != "" {
			return fmt.Errorf("version already set")
		}
		s.version = version
		return nil
	})
}
