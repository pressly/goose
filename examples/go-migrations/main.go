// This is custom goose binary with sqlite3 support only.

package main

import (
	"context"
	"database/sql"
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/pressly/goose/v4"

	_ "github.com/pressly/goose/v4/examples/go-migrations/migrations"
	_ "modernc.org/sqlite"
)

var (
	flags = flag.NewFlagSet("goose", flag.ExitOnError)
	dir   = flags.String("dir", "migrations", "directory with migration files")
)

func main() {
	ctx := context.Background()

	flags.Parse(os.Args[1:])
	args := flags.Args()
	if len(args) < 2 {
		flags.Usage()
		return
	}
	dbstring, command := args[0], args[1]
	// Open a connection to the database.
	db, err := sql.Open("sqlite", dbstring)
	if err != nil {
		log.Fatal(err)
	}
	// Create a goose provider.
	options := goose.DefaultOptions().
		SetDir(*dir)
	provider, err := goose.NewProvider(goose.DialectSQLite3, db, options)
	if err != nil {
		log.Fatal(err)
	}
	// Ping the database to ensure the connection is valid.
	if err := provider.Ping(ctx); err != nil {
		log.Fatal(err)
	}
	// Close the database connection when the program exits.
	defer func() {
		if err := provider.Close(); err != nil {
			log.Fatalf("goose: failed to close connection: %v\n", err)
		}
	}()
	// Run a goose command.
	switch command {
	case "up":
		results, err := provider.Up(ctx)
		if err != nil {
			log.Fatalf("goose %v: %v", command, err)
		}
		printResults(results)
	case "reset":
		results, err := provider.DownTo(ctx, 0)
		if err != nil {
			log.Fatalf("goose %v: %v", command, err)
		}
		printResults(results)
	default:
		log.Fatalf("goose: unknown command %q", command)
	}
}

func printResults(results []*goose.MigrationResult) {
	for _, res := range results {
		log.Printf("OK   %s (%s)", filepath.Base(res.Migration.Source), truncateDuration(res.Duration))
	}
}

func truncateDuration(d time.Duration) time.Duration {
	for _, v := range []time.Duration{
		time.Second,
		time.Millisecond,
		time.Microsecond,
	} {
		if d > v {
			return d.Round(v / time.Duration(100))
		}
	}
	return d
}
