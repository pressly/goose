package goosecli

import (
	"context"
	"errors"
	"flag"
	"path/filepath"
	"strconv"
	"time"

	"github.com/mfridman/cli"
	"github.com/pressly/goose/v3"
)

func defaultUsageFunc() func(c *cli.Command) string {
	return func(c *cli.Command) string {
		return newHelp().
			add("", shortHelpSection).
			add("USAGE", usageSection).
			add("COMMANDS", commandsSection).
			add("FLAGS", flagsSection).
			build(c)
	}
}

func dirFlag(f *flag.FlagSet) {
	f.String("dir", "", "Directory with migration files")
}

func dbStringFlag(f *flag.FlagSet) {
	f.String("dbstring", "", "Database connection string")
}

func tableFlag(f *flag.FlagSet) {
	f.String("table", goose.DefaultTablename, "Goose migration table name")
}

// commonConnectionFlags are flags that are required for most goose commands which interact with the
// database and open a connection.
func commonConnectionFlags(f *flag.FlagSet) {
	dirFlag(f)
	dbStringFlag(f)
	tableFlag(f)

	f.Duration("timeout", 0, "Maximum allowed duration for queries to run; e.g., 1h13m")

	// MySQL flags
	f.String("certfile", "", "File path to root CA's certificates in pem format (mysql only)")
	f.String("ssl-cert", "", "File path to SSL certificates in pem format (mysql only)")
	f.String("ssl-key", "", "File path to SSL key in pem format (mysql only)")
}

var downTo = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "down-to",
	Usage:     "goose down-to [flags] <version>",
	ShortHelp: "Roll back migrations down to, but not including, the specified version",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		commonConnectionFlags(f)
		f.Bool("no-versioning", false, "Apply migration commands with no versioning, in file order, from directory pointed to")
		f.Bool("json", false, "Output results in JSON format")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		printer := newPrinter(s.Stdout, defaultSeparator)

		useJSON := cli.GetFlag[bool](s, "json")

		if len(s.Args) == 0 {
			return errors.New("must supply a version to goose down-to")
		}
		version, err := strconv.ParseInt(s.Args[0], 10, 64)
		if err != nil {
			return errors.New("version must be a number")
		}

		provider, err := getProvider(
			s,
			goose.WithDisableVersioning(cli.GetFlag[bool](s, "no-versioning")),
		)
		if err != nil {
			return err
		}

		results, err := provider.DownTo(ctx, version)
		if err != nil {
			return err
		}

		return printResults(printer, results, useJSON)
	},
}

var down = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "down",
	Usage:     "goose down [flags]",
	ShortHelp: "Roll back the most recently applied migration",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		commonConnectionFlags(f)
		f.Bool("no-versioning", false, "Apply migration commands with no versioning, in file order, from directory pointed to")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		return errors.New("not implemented")
	},
}

var up = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "up",
	Usage:     "goose up [flags]",
	ShortHelp: "Apply all available migrations",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		commonConnectionFlags(f)
		f.Bool("allow-missing", false, "Applies missing (out-of-order) migrations")
		f.Bool("no-versioning", false, "Apply migration commands with no versioning, in file order, from directory pointed to")
		f.Bool("json", false, "Output results in JSON format")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		printer := newPrinter(s.Stdout, defaultSeparator)

		useJSON := cli.GetFlag[bool](s, "json")

		provider, err := getProvider(
			s,
			goose.WithDisableVersioning(cli.GetFlag[bool](s, "no-versioning")),
			goose.WithAllowOutofOrder(cli.GetFlag[bool](s, "allow-missing")),
		)
		if err != nil {
			return err
		}
		results, err := provider.Up(ctx)
		if err != nil {
			var partialErr *goose.PartialError
			if !errors.As(err, &partialErr) {
				return err
			}
			combined := partialErr.Applied
			combined = append(combined, partialErr.Failed)
			return printResults(printer, combined, useJSON)
		}
		return printResults(printer, results, useJSON)
	},
}

var upByOne = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "up-by-one",
	Usage:     "goose up-by-one [flags]",
	ShortHelp: "Apply the next available migration",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		commonConnectionFlags(f)
		f.Bool("allow-missing", false, "Applies missing (out-of-order) migrations")
		f.Bool("no-versioning", false, "Apply migration commands with no versioning, in file order, from directory pointed to")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		return errors.New("not implemented")
	},
}

var upTo = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "up-to",
	Usage:     "goose up-to [flags] <version>",
	ShortHelp: "Apply all available migrations up to, and including, the specified version",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		commonConnectionFlags(f)
		f.Bool("allow-missing", false, "Applies missing (out-of-order) migrations")
		f.Bool("no-versioning", false, "Apply migration commands with no versioning, in file order, from directory pointed to")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		return errors.New("not implemented")
	},
}

var status = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "status",
	Usage:     "goose status [flags]",
	ShortHelp: "List the status of all migrations",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		commonConnectionFlags(f)
		f.Bool("json", false, "Output results in JSON format")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		printer := newPrinter(s.Stdout, defaultSeparator)

		useJSON := cli.GetFlag[bool](s, "json")

		provider, err := getProvider(s)
		if err != nil {
			return err
		}

		results, err := provider.Status(ctx)
		if err != nil {
			return err
		}
		if useJSON {
			return printer.JSON(toMigrationStatus(results))
		}

		table := tableData{
			Headers: []string{"Migration name", "Applied At"},
		}
		for _, result := range results {
			status := "Pending"
			if result.State == goose.StateApplied {
				status = result.AppliedAt.Format(time.DateTime)
			}
			row := []string{
				filepath.Base(result.Source.Path),
				status,
			}
			table.Rows = append(table.Rows, row)
		}
		return printer.Table(table)
	},
}

var version = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "version",
	Usage:     "goose version [flags]",
	ShortHelp: "Print the current version of the database",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		commonConnectionFlags(f)
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		return errors.New("not implemented")
	},
}

var create = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "create",
	Usage:     "goose create [flags] <migration name>",
	ShortHelp: "Create a new migration file",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		dirFlag(f)
		f.String("s", "", "Use sequential numbering for new migrations")
		f.String("type", "sql", "Type of migration to create [sql,go]")
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		return errors.New("not implemented")
	},
}

var fix = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "fix",
	Usage:     "goose fix [flags]",
	ShortHelp: "Convert migration files to sequential order, while preserving timestamp ordering",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		dirFlag(f)
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		return errors.New("not implemented")
	},
}

var validate = &cli.Command{
	UsageFunc: defaultUsageFunc(),

	Name:      "validate",
	Usage:     "goose validate [flags]",
	ShortHelp: "Check migration files without running them",
	Flags: cli.FlagsFunc(func(f *flag.FlagSet) {
		dirFlag(f)
	}),
	Exec: func(ctx context.Context, s *cli.State) error {
		return errors.New("not implemented")
	},
}
