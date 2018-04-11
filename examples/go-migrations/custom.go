package main

import (
	// Import your migrations directory here
	_ "github.com/discovery-digital/goose/examples/go-migrations/migrations"
	// In your repo, replace "goose" line above with the path to goose:
	"github.com/discovery-digital/goose"
)

func main() {
	goose.Command()
}
