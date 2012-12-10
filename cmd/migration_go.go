package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

const (
	UpDecl      = "func Up(txn *sql.Tx) {"
	UpDeclFmt   = "func migration_%d_Up(txn *sql.Tx) {\n"
	DownDecl    = "func Down(txn *sql.Tx) {"
	DownDeclFmt = "func migration_%d_Down(txn *sql.Tx) {\n"
)

type TemplateData struct {
	Versions  []string
	DBDriver  string
	DBOpen    string
	Direction string
}

func directionStr(direction bool) string {
	if direction {
		return "Up"
	}
	return "Down"
}

//
// Run a .go migration.
//
// In order to do this, we copy a modified version of the
// original .go migration, and execute it via `go run` along
// with a main() of our own creation.
//
func runGoMigration(txn *sql.Tx, conf *DBConf, path string, version int, direction bool) (int, error) {

    // everything gets written to a temp file, and zapped afterwards
	d, e := ioutil.TempDir("", "goose")
	if e != nil {
		log.Fatal(e)
	}
	defer os.RemoveAll(d)

	td := &TemplateData{
		Versions:  []string{fmt.Sprintf("%d", version)},
		DBDriver:  conf.Driver,
		DBOpen:    conf.OpenStr,
		Direction: directionStr(direction),
	}
	main, e := writeTemplateToFile(filepath.Join(d, "goose_main.go"), td)
	if e != nil {
		log.Fatal(e)
	}

	outpath := filepath.Join(d, filepath.Base(path))
	if e = writeSubstituted(path, outpath, version); e != nil {
		log.Fatal(e)
	}

	cmd := exec.Command("go", "run", main, outpath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if e = cmd.Run(); e != nil {
		log.Fatal("`go run` failed: ", e)
	}

	return 0, nil
}

//
// a little cheesy, but do a simple text substitution on the contents of the
// migration script. this has 2 motivations:
//  * rewrite the package to 'main' so that we can run as part of `go run`
//  * namespace the Up() and Down() funcs so we can compile several
//    .go migrations into the same binary for execution
//
func writeSubstituted(inpath, outpath string, version int) error {

	fin, e := os.Open(inpath)
	if e != nil {
		return e
	}
	defer fin.Close()

	fout, e := os.OpenFile(outpath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if e != nil {
		return e
	}
	defer fout.Close()

	rw := bufio.NewReadWriter(bufio.NewReader(fin), bufio.NewWriter(fout))

	for {
		// XXX: could optimize the case in which we've already found
		// everything we're looking for, and just copy the rest in bulk

		line, _, e := rw.ReadLine()
		// XXX: handle isPrefix from ReadLine()

		if e != nil {
			if e != io.EOF {
				log.Fatal("failed to read from migration script:", e)
			}

			rw.Flush()
			break
		}

		lineStr := string(line)

		if strings.HasPrefix(lineStr, "package migration_") {
			if _, e := rw.WriteString("package main\n"); e != nil {
				return e
			}
			continue
		}

		if lineStr == UpDecl {
			up := fmt.Sprintf(UpDeclFmt, version)
			if _, e := rw.WriteString(up); e != nil {
				return e
			}
			continue
		}

		if lineStr == DownDecl {
			down := fmt.Sprintf(DownDeclFmt, version)
			if _, e := rw.WriteString(down); e != nil {
				return e
			}
			continue
		}

		// default case
		if _, e := rw.Write(line); e != nil {
			return e
		}
		if _, e := rw.WriteRune('\n'); e != nil {
			return e
		}
	}

	return nil
}

func writeTemplateToFile(path string, data *TemplateData) (string, error) {
	f, e := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if e != nil {
		return "", e
	}
	defer f.Close()

	e = tmpl.Execute(f, data)
	if e != nil {
		return "", e
	}

	return f.Name(), nil
}

//
// template for the main entry point to a go-based migration.
// this gets linked against the substituted versions of the user-supplied
// scripts in order to execute a migration via `go run`
//
var tmpl = template.Must(template.New("driver").Parse(`
package main

import (
	"database/sql"
	_ "github.com/bmizerany/pq"
	"log"
)

func main() {
	db, err := sql.Open("{{.DBDriver}}", "{{.DBOpen}}")
	if err != nil {
		log.Fatal("failed to open DB:", err)
	}
{{range .Versions}}

	// ----- migration {{ . }} -----
	txn, err := db.Begin()
	if err != nil {
		log.Fatal("db.Begin:", err)
	}

	migration_{{ . }}_{{$.Direction}}(txn)

	e := txn.Commit()
	if e != nil {
		log.Fatal("Commit() failed:", e)
	}{{end}}
}
`))
