package main

import (
	"fmt"
	"os"

	"github.com/pressly/goose/v4/internal/cli"
)

func main() {
	_ = normalizeDBString("", "", "", "", "")
	args := os.Args[1:]
	if err := cli.Run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
