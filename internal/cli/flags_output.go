package cli

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/charmbracelet/lipgloss"
	"github.com/peterbourgon/ff/v3/ffcli"
)

// It's inevitable that we'll have to update the flag output in the future. This is a simple way to
// do it based on our preferred style. We can add a new flag to the flagLookup map in flags.go and
// automatically have it show up in the help output.
//
// This ensures consistency across all commands.

type flagUsage struct {
	// short is the short description of the flag
	short string
	// defaultOption is the default value of the flag
	defaultOption string
	// availableOptions is a list of available options for the flag
	availableOptions []string
}

func defaultUsageFunc(examples []string) func(c *ffcli.Command) string {
	return func(c *ffcli.Command) string {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color("#F2451B"))

		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(c.LongHelp))
		b.WriteString("\n\n")
		b.WriteString(style.Render("USAGE"))
		b.WriteString("\n")
		b.WriteString("  " + c.ShortUsage)
		b.WriteString("\n\n")
		b.WriteString(style.Render("FLAGS"))
		b.WriteString("\n")
		if countFlags(c.FlagSet) > 0 {
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
		if len(examples) > 0 {
			b.WriteString("\n")
			b.WriteString(style.Render("EXAMPLES"))
			b.WriteString("\n")
			for _, e := range examples {
				b.WriteString("  " + e)
				b.WriteString("\n")
			}
		}
		return "\n" + strings.TrimSpace(b.String()) + "\n"
	}
}

func isBoolFlag(f *flag.Flag) bool {
	b, ok := f.Value.(interface {
		IsBoolFlag() bool
	})
	return ok && b.IsBoolFlag()
}

func countFlags(fs *flag.FlagSet) (n int) {
	fs.VisitAll(func(*flag.Flag) { n++ })
	return n
}
