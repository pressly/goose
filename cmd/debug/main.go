package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func main() {
	p, err := goose.NewProvider("sqlite", "cmd/debug/test.db", "cmd/debug/migrations")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	log.Fatal(p.Status(context.Background()))
}
