package main

import (
	"fmt"
	"os"

	"github.com/pressly/goose/v4/internal/cli"
)

func main() {
	args := os.Args[1:]
	if err := cli.Run(args); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR goose:", err)
		os.Exit(1)
	}
}
