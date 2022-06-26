package main

import (
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func main() {
	provider, err := goose.NewProvider("sqlite", "cmd/debug/test.db", "cmd/debug/migrations")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	_ = provider
}
