package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"text/template"
)

var commands = []*Command{
	upCmd,
	downCmd,
	statusCmd,
	createCmd,
}

func main() {

	flag.Usage = usage
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 || args[0] == "-h" {
		flag.Usage()
		return
	}

	var cmd *Command
	name := args[0]
	for _, c := range commands {
		if strings.HasPrefix(c.Name, name) {
			cmd = c
			break
		}
	}

	if cmd == nil {
		fmt.Printf("error: unknown command %q\n", name)
		flag.Usage()
		os.Exit(1)
	}

	cmd.Exec(args[1:])
}

func usage() {
	usageTmpl.Execute(os.Stdout, commands)
}

var usageTmpl = template.Must(template.New("usage").Parse(
	`goose is a database migration management system for Go projects.

Usage:
    goose [options] <subcommand> [subcommand options]

Commands:{{range .}}
    {{.Name | printf "%-10s"}} {{.Summary}}{{end}}
`))
