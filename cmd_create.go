package main

import (
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"text/template"
	"time"
)

var createCmd = &Command{
	Name:    "create",
	Usage:   "",
	Summary: "Create the scaffolding for a new migration",
	Help:    `create extended help here...`,
}

func createRun(cmd *Command, args ...string) {

	if len(args) < 1 {
		log.Fatal("goose create: migration name required")
	}

	migrationType := "go" // default to Go migrations
	if len(args) >= 2 {
		migrationType = args[1]
		if migrationType != "go" && migrationType != "sql" {
			log.Fatal("goose create: migration type must be 'go' or 'sql'")
		}
	}

	conf, err := MakeDBConf()
	if err != nil {
		log.Fatal(err)
	}

	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%v_%v.%v", timestamp, args[0], migrationType)
	err = os.MkdirAll(conf.MigrationsDir, 0777)
	if err != nil {
		log.Fatal(err)
	}
	fpath := path.Join(conf.MigrationsDir, filename)

	var tmpl *template.Template
	if migrationType == "sql" {
		tmpl = sqlMigrationScaffoldTmpl
	} else {
		tmpl = goMigrationScaffoldTmpl
	}

	n, e := writeTemplateToFile(fpath, tmpl, timestamp)
	if e != nil {
		log.Fatal(e)
	}

	a, e := filepath.Abs(n)
	if e != nil {
		log.Fatal(e)
	}

	fmt.Println("goose: created", a)
}

func init() {
	createCmd.Run = createRun
}

var goMigrationScaffoldTmpl = template.Must(template.New("driver").Parse(`
package main

import (
	"database/sql"
)

// Up is executed when this migration is applied
func Up_{{ . }}(txn *sql.Tx) {

}

// Down is executed when this migration is rolled back
func Down_{{ . }}(txn *sql.Tx) {

}
`))

var sqlMigrationScaffoldTmpl = template.Must(template.New("driver").Parse(`
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied


-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

`))
