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
	ctx := context.Background()
	p, err := goose.NewProvider("sqlite", "cmd/debug/test.db", "cmd/debug/migrations")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if err := p.Status(ctx); err != nil {
		log.Fatal(err)
	}
	if err := p.Up(ctx); err != nil {
		log.Fatal(err)
	}
}
