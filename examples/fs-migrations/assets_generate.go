// +build ignore

package main

import (
	"log"

	"github.com/shurcooL/vfsgen"

	"github.com/pressly/goose/examples/fs-migrations/postgres"
)

func main() {
	err := vfsgen.Generate(postgres.Migrations, vfsgen.Options{
		PackageName:  "postgres",
		BuildTags:    "!dev",
		VariableName: "Migrations",
	})

	if err != nil {
		log.Fatalln(err)
	}
}
