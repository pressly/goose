package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

var commands = []*Command{
	upCmd,
	downCmd,
}

func main() {

	// XXX: create a flag.Usage that dumps all commands
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
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
