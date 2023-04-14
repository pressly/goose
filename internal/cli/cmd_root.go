package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/peterbourgon/ff/v3/ffcli"
)

var (
	VERSION = ""
)

func newRootCmd(w io.Writer) (*ffcli.Command, *rootConfig) {
	config := &rootConfig{
		stdout: w,
	}
	fs := flag.NewFlagSet("goose", flag.ExitOnError)
	registerFlags(fs, config)
	fs.BoolVar(&config.version, "version", false, "")

	rootUsageOpt := &rootUsageOpt{
		supportedDatabases: []string{
			"postgres        mysql        sqlite3        clickhouse",
			"redshift        tidb         mssql          vertica",
		},
		environmentVariables: []string{EnvGooseDBString, EnvGooseDir, EnvGooseTable, EnvNoColor},
		examples: []string{
			`$ goose --dbstring="postgres://dbuser:password1@localhost:5432/testdb?sslmode=disable" status`,
			`$ goose --dbstring="mysql://user:password@/dbname?parseTime=true" status`,
			``,
			`$ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose status`,
			`$ GOOSE_DBSTRING="clickhouse://user:password@localhost:9000/clickdb" goose status`,
		},
		learnMore: []string{
			`Use 'goose <command> --help' for more information about a command.`,
			`Read the manual at https://pressly.github.io/goose/"`,
		},
	}
	root := &ffcli.Command{
		Name:       "goose",
		ShortUsage: "goose <command> [flags] [args...]",
		ShortHelp:  "goose is a database migration tool.",
		FlagSet:    fs,
		UsageFunc: func(c *ffcli.Command) string {
			return newRootUsage(c, rootUsageOpt)
		},
		Exec: func(ctx context.Context, args []string) error {
			if config.version {
				fmt.Fprintf(w, "goose version: %s\n", getVersion())
				return nil
			}
			if config.help {
				fs.Usage()
				return nil
			}
			return flag.ErrHelp
		},
	}
	return root, config
}

func getVersion() string {
	if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo != nil && VERSION == "" {
		VERSION = buildInfo.Main.Version
	}
	if VERSION == "" {
		VERSION = "(devel)"
	}
	return VERSION
}

type rootConfig struct {
	verbose bool
	useJSON bool
	noColor bool

	// version is a flag that prints the version of goose and exits.
	version bool
	help    bool

	// stdout is the output stream for the command. It is set to os.Stdout by default, but can be
	// overridden for testing.
	stdout io.Writer
}

// registerFlags registers the flag fields into the provided flag.FlagSet. This helper function
// allows subcommands to register the root flags into their flagsets, creating "global" flags that
// can be passed after any subcommand at the commandline.
func registerFlags(fs *flag.FlagSet, r *rootConfig) {
	fs.BoolVar(&r.verbose, "v", false, "")
	fs.BoolVar(&r.useJSON, "json", false, "")
	fs.BoolVar(&r.noColor, "no-color", false, "")
	fs.BoolVar(&r.help, "help", false, "")
}

type rootUsageOpt struct {
	supportedDatabases   []string
	environmentVariables []string
	examples             []string
	learnMore            []string
}

func newRootUsage(c *ffcli.Command, opt *rootUsageOpt) string {
	var b strings.Builder

	style := lipgloss.NewStyle().Foreground(lipgloss.Color(redColor))

	b.WriteString("\n")
	b.WriteString(c.ShortHelp)
	b.WriteString("\n\n")
	b.WriteString(style.Render("USAGE"))
	b.WriteString("\n")
	b.WriteString("  " + c.ShortUsage)
	b.WriteString("\n\n")
	b.WriteString(style.Render("COMMANDS"))
	b.WriteString("\n")
	tw := tabwriter.NewWriter(&b, 0, 2, 6, ' ', 0)
	sort.Slice(c.Subcommands, func(i, j int) bool {
		return c.Subcommands[i].Name < c.Subcommands[j].Name
	})
	for _, cmd := range c.Subcommands {
		fmt.Fprintf(tw, "  %s\t%s\n", cmd.Name, cmd.ShortHelp)
	}
	tw.Flush()
	b.WriteString("\n")
	b.WriteString(style.Render("SUPPORTED DATABASES"))
	b.WriteString("\n")
	for _, db := range opt.supportedDatabases {
		b.WriteString("  " + db + "\n")
	}
	b.WriteString("\n")
	if countFlags(c.FlagSet) > 0 {
		b.WriteString(style.Render("GLOBAL FLAGS"))
		b.WriteString("\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 6, ' ', 0)
		c.FlagSet.VisitAll(func(f *flag.Flag) {
			short := flagLookup[f.Name].short
			if flagLookup[f.Name].defaultOption != "" {
				if len(flagLookup[f.Name].availableOptions) > 0 {
					options := strings.Join(flagLookup[f.Name].availableOptions, ",")
					short += fmt.Sprintf(". Must be one of [%s]", options)
				}
				if isBoolFlag(f) {
					b, _ := strconv.ParseBool(flagLookup[f.Name].defaultOption)
					short += fmt.Sprintf(" (default: %t)", b)
				} else {
					short += fmt.Sprintf(" (default: %q)", flagLookup[f.Name].defaultOption)
				}
			}
			// TODO(mf): handle overflow scenario where short is too long and spills over to the
			// next column
			fmt.Fprintf(tw, "  --%s\t%s\n", f.Name, short)
		})
		tw.Flush()
	}
	if len(opt.environmentVariables) > 0 {
		b.WriteString("\n")
		b.WriteString(style.Render("ENVIRONMENT VARIABLES"))
		b.WriteString("\n")
		tw := tabwriter.NewWriter(&b, 0, 2, 6, ' ', 0)
		for _, e := range opt.environmentVariables {
			desc, ok := envLookup[e]
			if ok && desc != "" {
				fmt.Fprintf(tw, "  %s\t%s\n", e, desc)
			}
		}
		tw.Flush()
	}
	if len(opt.examples) > 0 {
		b.WriteString("\n")
		b.WriteString(style.Render("EXAMPLES"))
		b.WriteString("\n")
		for _, e := range opt.examples {
			b.WriteString("  " + e + "\n")
		}
	}
	if len(opt.learnMore) > 0 {
		b.WriteString("\n")
		b.WriteString(style.Render("LEARN MORE"))
		b.WriteString("\n")
		for _, l := range opt.learnMore {
			b.WriteString("  " + l + "\n")
		}
	}
	return "\n" + strings.TrimSpace(b.String()) + "\n"
}

const (
	rootUsageHelp = `
A database migration tool.

USAGE
  goose <command> [flags] [args...]

COMMANDS
  create          Create a new .go or .sql migration file
  down            Migrate database down to the previous version
  down-to         Migrate database down to, but not including, a specific version
  env             Print environment variables
  fix             Apply sequential numbering to existing timestamped migrations
  redo            Roll back the last appied migration and re-apply it
  status          List applied and pending migrations
  up              Migrate database to the most recent version
  up-by-one       Migrate database up by one version
  up-to           Migrate database up to, and including, a specific version
  validate        Validate migration files in the migrations directory
  version         Print the current version of the database

SUPPORTED DATABASES
  postgres        mysql        sqlite3
  redshift        tidb         mssql
  clickhouse      vertica      

GLOBAL FLAGS
  --help              Display help
  --json              Format output as JSON
  --no-color          Disable color output
  --v                 Turn on verbose mode
  --version           Display the version of goose currently installed

ENVIRONMENT VARIABLES
  GOOSE_DBSTRING      Database connection string, lower priority than --dbstring
  GOOSE_DIR           Directory with migration files, lower priority than --dir
  GOOSE_TABLE         Database table name, lower priority than --table (default "goose_db_version")
  NO_COLOR            Disable color output

EXAMPLES
  $ goose --dbstring="postgres://dbuser:password1@localhost:5432/testdb?sslmode=disable" status
  $ goose --dbstring="mysql://user:password@/dbname?parseTime=true" status

  $ GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING="sqlite:./test.db" goose status
  $ GOOSE_DBSTRING="clickhouse://user:password@localhost:9000/clickdb" goose status

LEARN MORE
  Use 'goose <command> --help' for more information about a command.
  Read the manual at https://pressly.github.io/goose/
`
)
