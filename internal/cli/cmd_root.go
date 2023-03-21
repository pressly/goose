package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
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

func newRootCmd() (*ffcli.Command, *rootConfig) {
	config := &rootConfig{}
	fs := flag.NewFlagSet("goose", flag.ExitOnError)
	config.registerFlags(fs)
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
			return newRootUsage(c, rootUsageOpt, config.noColor)
		},
		Exec: func(ctx context.Context, args []string) error {
			if config.version {
				fmt.Fprintf(os.Stdout, "goose version: %s\n", getVersion())
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
}

// registerFlags registers the flag fields into the provided flag.FlagSet. This helper function
// allows subcommands to register the root flags into their flagsets, creating "global" flags that
// can be passed after any subcommand at the commandline.
func (c *rootConfig) registerFlags(fs *flag.FlagSet) {
	fs.BoolVar(&c.verbose, "v", false, "")
	fs.BoolVar(&c.useJSON, "json", false, "")
	fs.BoolVar(&c.noColor, "no-color", false, "")
	fs.BoolVar(&c.help, "help", false, "")
}

type rootUsageOpt struct {
	supportedDatabases   []string
	environmentVariables []string
	examples             []string
	learnMore            []string
}

func newRootUsage(c *ffcli.Command, opt *rootUsageOpt, noColor bool) string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(redColor))
	render := func(s string) string {
		if noColor {
			return s
		}
		return style.Render(s)
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(c.ShortHelp)
	b.WriteString("\n\n")
	b.WriteString(render("USAGE"))
	b.WriteString("\n")
	b.WriteString("  " + c.ShortUsage)
	b.WriteString("\n\n")
	b.WriteString(render("COMMANDS"))
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
	b.WriteString(render("SUPPORTED DATABASES"))
	b.WriteString("\n")
	for _, db := range opt.supportedDatabases {
		b.WriteString("  " + db + "\n")
	}
	b.WriteString("\n")
	if countFlags(c.FlagSet) > 0 {
		b.WriteString(render("GLOBAL FLAGS"))
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
		b.WriteString(render("ENVIRONMENT VARIABLES"))
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
		b.WriteString(render("EXAMPLES"))
		b.WriteString("\n")
		for _, e := range opt.examples {
			b.WriteString("  " + e + "\n")
		}
	}
	if len(opt.learnMore) > 0 {
		b.WriteString("\n")
		b.WriteString(render("LEARN MORE"))
		b.WriteString("\n")
		for _, l := range opt.learnMore {
			b.WriteString("  " + l + "\n")
		}
	}
	return "\n" + strings.TrimSpace(b.String()) + "\n"
}
