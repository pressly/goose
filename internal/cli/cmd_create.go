package cli

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/peterbourgon/ff/v3"
	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/pressly/goose/v4"
)

type createCmd struct {
	root *rootConfig

	sequential bool
	noTx       bool
}

func newCreateCmd(root *rootConfig) *ffcli.Command {
	c := createCmd{root: root}
	fs := flag.NewFlagSet("goose create", flag.ExitOnError)
	fs.BoolVar(&c.sequential, "s", false, "use sequential versions")
	fs.BoolVar(&c.noTx, "no-tx", false, "do not wrap migration in a transaction")
	root.registerFlags(fs)

	return &ffcli.Command{
		Name:       "create",
		ShortUsage: "goose [flags] create [sql|go] <name> ",
		LongHelp:   "",
		ShortHelp:  "",
		Exec:       c.Exec,
		FlagSet:    fs,
		Options: []ff.Option{
			ff.WithEnvVarPrefix("GOOSE"),
		},
	}
}

func (c *createCmd) Exec(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("create requires 2 arguments: [sql|go] <name>\n\nExample: goose create sql add users table")
	}

	var migrationType goose.MigrationType
	switch strings.ToLower(args[0]) {
	case "go":
		migrationType = goose.MigrationTypeGo
	case "sql":
		migrationType = goose.MigrationTypeSQL
	default:
		return fmt.Errorf("invalid migration type: first argument must be one of [sql|go]\n\nExample: goose create sql add users table")
	}

	name := strings.Join(args[1:], " ")
	filename, err := goose.Create(c.root.dir, migrationType, name, &goose.CreateOptions{
		Sequential: c.sequential,
		NoTx:       c.noTx,
	})
	if err != nil {
		return err
	}
	fmt.Printf("Created: %s\n", filename)
	return nil
}
