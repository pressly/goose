package main

import (
	"fmt"
	"os"

	"github.com/pressly/goose/v3"
)

func main() {
	dbString := "postgresql://dbuser:password123@localhost:5432/bestofgodb?sslmode=disable"
	_, err := goose.NewProvider("postgres", dbString, "cmd/debug/migrations")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
