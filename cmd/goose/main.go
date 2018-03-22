package main

import (
	_ "migrations"
	"goose"
	// In your repo, replace the line above with the line below:
	//"github.com/pressly/goose"
)

func main() {
	goose.Command()
}
