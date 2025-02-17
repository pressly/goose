package goosecli

import (
	"cmp"
	"flag"
	"fmt"
	"slices"
	"strings"

	"github.com/mfridman/cli"
	"github.com/pressly/goose/v3"
)

type helpSection struct {
	title   string
	content func(*cli.Command) string
}

type help struct {
	sections []helpSection
}

func newHelp() *help {
	return &help{}
}

func (h *help) add(title string, content func(*cli.Command) string) *help {
	h.sections = append(h.sections, helpSection{title, content})
	return h
}

func (h *help) build(c *cli.Command) string {
	var sb strings.Builder
	for _, section := range h.sections {
		if section.content != nil {
			if content := section.content(c); strings.TrimSpace(content) != "" {
				if section.title != "" {
					sb.WriteString(render(section.title) + "\n")
				}
				sb.WriteString(content)
			}
		}
	}
	return strings.TrimSpace(sb.String()) + "\n"
}

func shortHelpSection(c *cli.Command) string {
	return c.ShortHelp + "\n\n"
}

func usageSection(c *cli.Command) string {
	return fmt.Sprintf("  %s\n\n", c.Usage)
}

func commandsSection(c *cli.Command) string {
	maxLen := 0
	for _, cmd := range c.SubCommands {
		maxLen = max(maxLen, len(cmd.Name))
	}

	// Add commands with dynamic padding
	var sb strings.Builder
	for _, cmd := range c.SubCommands {
		padding := strings.Repeat(" ", maxLen-len(cmd.Name)+2) // +2 for minimal spacing
		sb.WriteString(fmt.Sprintf("  %s%s%s\n", cmd.Name, padding, cmd.ShortHelp))
	}

	return sb.String() + "\n"
}

func flagsSection(c *cli.Command) string {
	if c.Flags == nil {
		return "\n"
	}
	maxLen := 0
	// First pass to find the longest flag name + value
	c.Flags.VisitAll(func(f *flag.Flag) {
		maxLen = max(maxLen, len(f.Name))
	})
	var all []*flag.Flag
	c.Flags.VisitAll(func(f *flag.Flag) {
		all = append(all, f)
	})
	// Sort flags by name
	slices.SortFunc(all, func(a, b *flag.Flag) int {
		return cmp.Compare(a.Name, b.Name)
	})

	// Second pass to write formatted flags
	var sb strings.Builder
	for _, f := range all {
		flagText := "--" + f.Name
		padding := strings.Repeat(" ", maxLen-len(flagText)+4) // +4 for extra spacing
		sb.WriteString(fmt.Sprintf("  %s%s%s\n", flagText, padding, f.Usage))
	}

	return sb.String() + "\n"
}

func databasesSection(_ *cli.Command) string {
	databases := []string{
		"postgres", "mysql", "sqlite3", "clickhouse",
		"redshift", "tidb", "mssql", "vertica",
	}
	var sb strings.Builder
	maxWidth := 12
	dbPerLine := 4
	for i := 0; i < len(databases); i += dbPerLine {
		sb.WriteString("  ")
		end := min(i+dbPerLine, len(databases))
		for j := i; j < end; j++ {
			db := databases[j]
			padding := strings.Repeat(" ", maxWidth-len(db))
			sb.WriteString(db + padding)
		}
		sb.WriteString("\n")
	}

	return sb.String() + "\n"
}

func envVarsSection(_ *cli.Command) string {
	lines := []struct{ name, desc string }{
		{"GOOSE_DBSTRING", "Database connection string"},
		{"GOOSE_DIR", "Directory with migration files"},
		{"GOOSE_TABLE", fmt.Sprintf("Migrations table name (default: %s)", goose.DefaultTablename)},
		{"NO_COLOR", "Disable color output"},
	}
	maxLen := 0
	for _, line := range lines {
		maxLen = max(maxLen, len(line.name))
	}

	var sb strings.Builder
	for _, line := range lines {
		padding := strings.Repeat(" ", maxLen-len(line.name)+2)
		sb.WriteString(fmt.Sprintf("  %s%s%s\n", line.name, padding, line.desc))
	}

	return sb.String() + "\n"
}

func learnMoreSection(_ *cli.Command) string {
	lines := []string{
		"  Use 'goose <command> --help' for more information about a command",
		"  Read the docs at https://pressly.github.io/goose/",
	}

	return strings.Join(lines, "\n") + "\n"
}
