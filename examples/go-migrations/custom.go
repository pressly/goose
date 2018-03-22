package main

import (
	// Import your migrations directory here
	_ "goose/examples/go-migrations/migrations"
	// In your repo, replace "goose" line above with the path to goose:
	"github.com/pressly/goose"
)

func main() {
	goose.Command()
}
