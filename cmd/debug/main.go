package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

func main() {
	flag.Parse()

	ctx := context.Background()

	db, err := sql.Open("sqlite", "sqlite.db")
	if err != nil {
		log.Fatal(err)
	}
	p, err := goose.NewProvider(goose.DialectSQLite3, db, os.DirFS("testdata/migrations"), goose.WithVerbose(true))
	if err != nil {
		log.Fatal(err)
	}
	switch flag.Arg(0) {
	case "up":
		fmt.Printf(">>> running: up\n\n")
		_, err = p.Up(ctx)
		if err != nil {
			log.Fatal(err)
		}
	case "up-to":
		fmt.Printf(">>> running: up-to %s\n\n", flag.Arg(1))
		n, err := strconv.Atoi(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		_, err = p.UpTo(ctx, int64(n))
		if err != nil {
			log.Fatal(err)
		}
	case "up-by-one":
		fmt.Printf(">>> running: up-by-one\n\n")
		_, err = p.UpByOne(ctx)
		if err != nil {
			log.Fatal(err)
		}
	case "down":
		fmt.Printf(">>> running: down\n\n")
		_, err = p.Down(ctx)
		if err != nil {
			log.Fatal(err)
		}
	case "down-to":
		fmt.Printf(">>> running: down-to %s\n\n", flag.Arg(1))
		n, err := strconv.Atoi(flag.Arg(1))
		if err != nil {
			log.Fatal(err)
		}
		_, err = p.DownTo(ctx, int64(n))
		if err != nil {
			log.Fatal(err)
		}
	}
}
