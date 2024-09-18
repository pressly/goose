package cli

import (
	"bytes"
	"os"
	"strconv"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

const (
	redColor = "#cc0000"
)

// additionalSections contains additional help sections for specific commands.
var additionalSections = map[string][]ffhelp.Section{
	"status": {
		{
			Title: "EXAMPLES",
			Lines: []string{
				`goose status --dir=migrations --dbstring=sqlite:./test.db`,
				`GOOSE_DIR=migrations GOOSE_DBSTRING=sqlite:./test.db goose status`,
			},
			LinePrefix: ffhelp.DefaultLinePrefix,
		},
	},
}

func createHelp(cmd *ff.Command) ffhelp.Help {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(redColor))
	render := func(s string) string {
		// TODO(mf): should we also support a global flag to disable color?
		if val := os.Getenv("NO_COLOR"); val != "" {
			if ok, err := strconv.ParseBool(val); err == nil && ok {
				return s
			}
		}
		return style.Render(s)
	}
	if selected := cmd.GetSelected(); selected != nil {
		cmd = selected
	}
	// For the root command, we're going to print a custom help message.
	if cmd.Name == "goose" {
		return rootHelp(cmd, render)
	}
	// For all other commands, we're going to print the default help message.
	var help ffhelp.Help

	if cmd.LongHelp != "" {
		section := ffhelp.NewUntitledSection(cmd.LongHelp)
		help = append(help, section)
	}

	title := cmd.Name
	if cmd.ShortHelp != "" {
		title = title + " -- " + cmd.ShortHelp
	}
	help = append(help, ffhelp.NewSection(render("COMMAND"), title))

	if cmd.Usage != "" {
		help = append(help, ffhelp.NewSection(render("USAGE"), cmd.Usage))
	}

	if len(cmd.Subcommands) > 0 {
		section := ffhelp.NewSubcommandsSection(cmd.Subcommands)
		section.Title = render(section.Title)
		help = append(help, section)
	}

	for _, section := range ffhelp.NewFlagsSections(cmd.Flags) {
		section.Title = render(section.Title)
		help = append(help, section)
	}
	if sections, ok := additionalSections[cmd.Name]; ok {
		for _, section := range sections {
			section.Title = render(section.Title)
			help = append(help, section)
		}
	}

	return help
}

func rootHelp(cmd *ff.Command, render func(s string) string) ffhelp.Help {
	var help ffhelp.Help

	section := ffhelp.NewUntitledSection("A database migration tool. Supports SQL migrations and Go functions.")
	help = append(help, section)

	section = ffhelp.NewSection(render("USAGE"), "goose <command> [flags] [args...]")
	help = append(help, section)

	section = ffhelp.NewSubcommandsSection(cmd.Subcommands)
	section.Title = render("COMMANDS")
	help = append(help, section)

	section = ffhelp.NewUntitledSection(render("SUPPORTED DATABASES"))
	for _, s := range []string{
		"postgres 	  mysql        sqlite3        clickhouse",
		"redshift 	  tidb         mssql          vertica",
	} {
		section.Lines = append(section.Lines, ffhelp.DefaultLinePrefix+s)
	}
	help = append(help, section)

	section = ffhelp.NewUntitledSection(render("ENVIRONMENT VARIABLES"))
	keys := []struct {
		name        string
		description string
	}{
		{"GOOSE_DBSTRING", "Database connection string, lower priority than --dbstring"},
		{"GOOSE_DIR", "Directory with migration files, lower priority than --dir"},
		{"GOOSE_TABLE", "Database table name, lower priority than --table (default goose_db_version)"},
		{"NO_COLOR", "Disable color output"},
	}
	buf := bytes.NewBuffer(nil)
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	for _, v := range keys {
		_, _ = tw.Write([]byte(ffhelp.DefaultLinePrefix + v.name + "\t" + v.description + "\n"))
	}
	tw.Flush()
	section.Lines = append(section.Lines, buf.String())
	help = append(help, section)

	// section = ffhelp.NewUntitledSection("EXAMPLES")
	// for _, s := range []string{
	// 	"goose status --dbstring=\"postgres://dbuser:password1@localhost:5433/testdb?sslmode=disable\" --dir=./examples/sql-migrations",
	// 	"GOOSE_DIR=./examples/sql-migrations GOOSE_DBSTRING=\"sqlite:./test.db\" goose status",
	// } {
	// 	section.Lines = append(section.Lines, s)
	// }
	// help = append(help, section)

	section = ffhelp.NewUntitledSection(render("LEARN MORE"))
	section.Lines = append(section.Lines, ffhelp.DefaultLinePrefix+"Use 'goose <command> --help' for more information about a command")
	section.Lines = append(section.Lines, ffhelp.DefaultLinePrefix+"Read the docs at https://pressly.github.io/goose/")
	help = append(help, section)

	return help
}
