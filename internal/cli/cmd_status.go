package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/peterbourgon/ff/v4"
	"github.com/pressly/goose/v3"
)

type cmdStatus struct {
	state *state
	fs    *ff.FlagSet

	// flags
	dir       string
	dbstring  string
	tablename string
	useJSON   bool
}

// TODO(mf): there is something not very ergonomic about how all this works. Will need to think
// about how to improve this and file an issue upstream. I wish the default could be set here,
// instead of in the flag definition.
func mustFlag(fs *ff.FlagSet, cfg ff.FlagConfig) {
	if _, err := fs.AddFlag(cfg); err != nil {
		panic(err)
	}
}

func newStatusCommand(state *state) (*ff.Command, error) {
	c := cmdStatus{
		state: state,
		fs:    ff.NewFlagSet("status"),
	}
	// Mandatory flags
	mustFlag(c.fs, newDirFlag(&c.dir))
	mustFlag(c.fs, newDBStringFlag(&c.dbstring))
	// Optional flags
	mustFlag(c.fs, newTablenameFlag(&c.tablename))
	mustFlag(c.fs, newJSONFlag(&c.useJSON))

	return &ff.Command{
		Name:      "status",
		Usage:     "status [flags]",
		ShortHelp: "List the status of all migrations",
		LongHelp:  strings.TrimSpace(statusLongHelp),
		Flags:     c.fs,
		Exec:      c.exec,
	}, nil
}

const (
	statusLongHelp = `
List the status of all migrations, comparing the current state of the database with the migrations
available in the filesystem. If a migration is applied to the database, it will be listed with the
timestamp it was applied, otherwise it will be listed as "Pending".
`
)

func (c *cmdStatus) exec(ctx context.Context, args []string) error {
	p, err := c.state.initProvider(c.dir, c.dbstring, c.tablename)
	if err != nil {
		return err
	}
	results, err := p.Status(ctx)
	if err != nil {
		return err
	}
	if c.useJSON {
		return c.state.writeJSON(convertMigrationStatus(results))
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', tabwriter.TabIndent)
	defer tw.Flush()
	fmtPattern := "%v\t%v\n"
	fmt.Fprintf(tw, fmtPattern, "Migration name", "Applied At")
	fmt.Fprintf(tw, fmtPattern, "──────────────", "──────────")
	for _, result := range results {
		t := "Pending"
		if result.State == goose.StateApplied {
			t = result.AppliedAt.Format(time.DateTime)
		}
		fmt.Fprintf(tw, fmtPattern, filepath.Base(result.Source.Path), t)
	}
	return nil
}
