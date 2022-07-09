// This is custom goose binary with sqlite3 support only.

package main

import (
	"flag"
	"os"

	_ "modernc.org/sqlite"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", ".", "directory with migration files")
)

func main() {
	flags.Parse(os.Args[1:])
	args := flags.Args()

	if len(args) < 3 {
		flags.Usage()
		return
	}

	dbstring, command := args[1], args[2]

	_ = dir
	_, _ = dbstring, command

	// TODO(mf): write new example.
}
