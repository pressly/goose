package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/pressly/goose"

	// Import package so migrations are registered via init()
	_ "migrations"
)

func main() {
	driver, dbstring, command := os.Args[1], os.Args[2], os.Args[3]
	if len(os.Args) != 3 {
		log.Fatalf("unexpected arguments. please use ./binary [driver] [conn_string] [command]")
	}

	db, err := sql.Open(driver, dbstring)
	if err != nil {
		log.Fatalf("goose: failed to open DB: %v\n", err)
	}

	err := goose.Registered().Run(command, db)
	if err != nil {
		log.Fatalln(err)
	}

}
