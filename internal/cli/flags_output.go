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

const (
	redColor = "#cc0000"
)

// It's inevitable that we'll have to update the flag output in the future. This is a simple way to
// do it based on our preferred style. We can add a new flag to the flagLookup map in flags.go and
// automatically have it show up in the help output.
//
// This ensures consistency across all commands.

type usageOpt struct {
	envs     []string
	examples []string
}

func newUsageFunc(opt *usageOpt) func(c *ffcli.Command) string {
	return func(c *ffcli.Command) string {
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(redColor))
		noColorPtr := c.FlagSet.Lookup("no-color")
		render := func(s string) string {
			if noColorPtr != nil {
				ok, err := strconv.ParseBool(noColorPtr.Value.String())
				if err == nil && ok {
					return s
				}
			}
			return style.Render(s)
		}

		if opt == nil {
			opt = &usageOpt{}
		}

		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(c.LongHelp))
		b.WriteString("\n\n")
		b.WriteString(render("USAGE"))
		b.WriteString("\n")
		b.WriteString("  " + strings.TrimSpace(c.ShortUsage))
		b.WriteString("\n")
		if countFlags(c.FlagSet) > 0 {
			b.WriteString("\n")
			b.WriteString(render("FLAGS"))
			b.WriteString("\n")
			tw := tabwriter.NewWriter(&b, 0, 2, 6, ' ', 0)
			c.FlagSet.VisitAll(func(f *flag.Flag) {
				short := flagLookup[f.Name].short
				defaultValue := coalesce(flagLookup[f.Name].defaultOption, f.DefValue)
				if defaultValue != "" {
					if len(flagLookup[f.Name].availableOptions) > 0 {
						options := strings.Join(flagLookup[f.Name].availableOptions, ",")
						short += fmt.Sprintf(". Must be one of [%s]", options)
					}
					// TODO(mf): if a bool is set to true in the BoolVar call, we need to
					// handle the default value differently. I.e., expose the true value
					// as the default.
					//
					// 🦄 This is a bit gross, but it works for now.
					if isBoolFlag(f) {
						b, _ := strconv.ParseBool(defaultValue)
						short += fmt.Sprintf(" (default: %t)", b)
					} else {
						short += fmt.Sprintf(" (default: %q)", flagLookup[f.Name].defaultOption)
					}
				}
				// TODO(mf): handle overflow scenario where short is too long and spills over to the
				// next line. Ideally we wrap the text.
				fmt.Fprintf(tw, "  --%s\t%s\n", f.Name, short)
			})
			tw.Flush()
		}

		if len(opt.envs) > 0 {
			b.WriteString("\n")
			b.WriteString(render("ENVIRONMENT VARIABLES"))
			b.WriteString("\n")
			tw := tabwriter.NewWriter(&b, 0, 2, 6, ' ', 0)
			for _, e := range opt.envs {
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
