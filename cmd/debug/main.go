package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/pressly/goose/v4"
	_ "github.com/pressly/goose/v4/cmd/debug/migrations"

	// _ "github.com/pressly/goose/v4/cmd/debug/migrationsb"
	_ "modernc.org/sqlite"
)

func main() {
	ctx := context.Background()
	db, err := sql.Open("sqlite", "cmd/debug/test.db")
	if err != nil {
		log.Fatal(err)
	}
	p, err := goose.NewProvider(goose.DialectSqlite, db, "cmd/debug/migrations", nil)
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
