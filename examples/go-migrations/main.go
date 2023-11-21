// This is custom goose binary with sqlite3 support only.

package main

import (
	"github.com/pressly/goose/v3"
	"os"
)

func main() {
	os.Args = append([]string{os.Args[0], "sqlite3"}, os.Args[1:]...)
	goose.Main()
}
