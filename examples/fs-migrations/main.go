package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"

	"github.com/pressly/goose"

	"github.com/pressly/goose/examples/fs-migrations/postgres"
	// import migrations in order to register any .go migrations
	_ "github.com/pressly/goose/examples/fs-migrations/postgres/migrations"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "fs_migrate"
)

func main() {
	dbConnectionString := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", dbConnectionString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	g := goose.New("./postgres/migrations", db, goose.WithFileSystem(postgres.Migrations))
	if err := g.Up(); err != nil {
		log.Fatal(err)
	}
}
