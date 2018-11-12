package main

import (
	"flag"
	"log"
	"os"

	"github.com/pressly/goose"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", ".", "directory with migration files")
)

func main() {
	flags.Usage = usage
	flags.Parse(os.Args[1:])

	args := flags.Args()
	switch args[0] {
	case "create":
		if err := goose.Run("create", nil, *dir, args[1:]...); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	case "fix":
		if err := goose.Run("fix", nil, *dir); err != nil {
			log.Fatalf("goose run: %v", err)
		}
		return
	}
}

func usage() {
	log.Print(usagePrefix)
	flags.PrintDefaults()
	log.Print(usageCommands)
}

var (
	usagePrefix = `Usage: goose [OPTIONS] COMMAND

Examples:
    goose create add_some_column sql
    goose create fetch_user_data go
    goose fix -dir=data

Options:
`

	usageCommands = `
Commands:
		create NAME [sql|go] Creates new migration file with the current timestamp
		fix                  Apply sequential ordering to migrations
`
)
