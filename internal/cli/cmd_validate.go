package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/migrationstats"
	"github.com/pressly/goose/v4/internal/migrationstats/migrationstatsos"
)

type validateCmd struct {
	root *rootConfig

	dir              string
	excludeFilenames stringSet
}

func newValidateCmd(root *rootConfig) *ffcli.Command {
	c := &validateCmd{root: root}
	fs := flag.NewFlagSet("goose validate", flag.ExitOnError)
	root.registerFlags(fs)
	fs.StringVar(&c.dir, "dir", "", "")
	fs.Var(&c.excludeFilenames, "exclude", "")

	return &ffcli.Command{
		Name:       "validate",
		ShortUsage: "goose validate [flags]",
		ShortHelp:  "Check migration files without running them",
		LongHelp:   validateCmdLongHelp,
		FlagSet:    fs,
		UsageFunc:  newUsageFunc(nil),
		Exec:       c.Exec,
	}
}

func (c *validateCmd) Exec(ctx context.Context, args []string) error {
	if len(args) > 0 {
		return errors.New("validate: invalid usage (expected no arguments). See `goose validate --help` for more information")
	}
	dir := coalesce(c.dir, GOOSE_DIR)
	if dir == "" {
		return fmt.Errorf("goose validate requires a migrations directory: %w", errNoDir)
	}
	sources, err := goose.Collect(dir, c.excludeFilenames)
	if err != nil {
		return err
	}
	if len(sources) == 0 {
		return fmt.Errorf("no migration sources found in %q", dir)
	}
	filenames := make([]string, 0, len(sources))
	for _, src := range sources {
		filenames = append(filenames, src.Fullpath)
	}
	fileWalker := migrationstatsos.NewFileWalker(filenames...)
	stats, err := migrationstats.GatherStats(fileWalker, false)
	if err != nil {
		return err
	}
	// TODO(mf): we should introduce a --debug flag, which allows printing
	// more internal debug information and leave verbose for additional information.
	if !c.root.verbose {
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	fmtPattern := "%v\t%v\t%v\t%v\t%v\t\n"
	fmt.Fprintf(w, fmtPattern, "Type", "Txn", "Up", "Down", "Name")
	fmt.Fprintf(w, fmtPattern, "────", "───", "──", "────", "────")
	for _, m := range stats {
		txnStr := "✔"
		if !m.Tx {
			txnStr = "✘"
		}
		fmt.Fprintf(w, fmtPattern,
			strings.TrimPrefix(filepath.Ext(m.FileName), "."),
			txnStr,
			m.UpCount,
			m.DownCount,
			filepath.Base(m.FileName),
		)
	}
	return w.Flush()
}

const validateCmdLongHelp = `
Check migration files without running them. Supports both SQL and Go migrations.
`
