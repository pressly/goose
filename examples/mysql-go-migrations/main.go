// This is a custom goose binary with MySQL support.
//
// Build it with:
//
//	go build -o goose-mysql *.go
//
// Then run migrations against a MySQL database:
//
//	./goose-mysql mysql "user:password@tcp(localhost:3306)/dbname?parseTime=true&multiStatements=true" status
//	./goose-mysql mysql "user:password@tcp(localhost:3306)/dbname?parseTime=true&multiStatements=true" up

package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/pressly/goose/v3"
	_ "github.com/go-sql-driver/mysql"
)

var (
	flags = flag.NewFlagSet("goose-mysql", flag.ExitOnError)
	dir   = flags.String("dir", ".", "directory with migration files")
)

func main() {
	if err := flags.Parse(os.Args[1:]); err != nil {
		log.Fatalf("goose-mysql: failed to parse flags: %v", err)
	}
	args := flags.Args()

	if len(args) < 3 {
		flags.Usage()
		return
	}

	// args[0] is the driver (mysql), args[1] is the dbstring, args[2] is the command
	driver, dbstring, command := args[0], args[1], args[2]

	if driver != "mysql" {
		log.Fatalf("goose-mysql: this binary only supports the 'mysql' driver, got %q", driver)
	}

	db, err := goose.OpenDBWithDriver(driver, dbstring)
	if err != nil {
		log.Fatalf("goose-mysql: failed to open DB: %v", err)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Fatalf("goose-mysql: failed to close DB: %v", err)
		}
	}()

	arguments := []string{}
	if len(args) > 3 {
		arguments = append(arguments, args[3:]...)
	}

	ctx := context.Background()
	if err := goose.RunContext(ctx, command, db, *dir, arguments...); err != nil {
		log.Fatalf("goose-mysql %v: %v", command, err)
	}
}