package e2e

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/pressly/goose/v3/internal/check"
	"github.com/pressly/goose/v3/internal/testdb"
)

const (
	dialectPostgres = "postgres"
	dialectMySQL    = "mysql"
)

// Flags.
var (
	debug = flag.Bool(
		"debug",
		false,
		"Debug traps the test suite: useful for debugging running containers",
	)
	dialect = flag.String(
		"dialect",
		dialectPostgres,
		"Dialect defines which docker container to run tests against (default: postgres)",
	)
	// bindPort is useful if you want to pin a database port instead of relying
	// on the randomly assigned one from Docker. It is mainly used for debugging
	// locally and will normally be set to 0.
	bindPort = flag.Int(
		"port",
		0,
		"Port is an optional bind port. Left empty will let Docker assign a random port (recommended)",
	)
)

var (
	// migrationsDir is a global that points to a ./testdata/{dialect}/migrations folder.
	// It is set in TestMain based on the current dialect.
	migrationsDir = ""
	// seedDir is similar to migrationsDir but contains seed data
	seedDir = ""

	// known tables are the tables (including goose table) created by
	// running all migration files. If you add a table, make sure to
	// add to this list and keep it in order.
	knownTables = []string{
		"goose_db_version",
		"issues",
		"owners",
		"repos",
		"stargazers",
	}
)

func TestMain(m *testing.M) {
	flag.Parse()

	switch *dialect {
	case dialectPostgres, dialectMySQL:
	default:
		log.Printf("dialect not supported: %q", *dialect)
		os.Exit(1)
	}
	migrationsDir = filepath.Join("testdata", *dialect, "migrations")
	seedDir = filepath.Join("testdata", *dialect, "seed")

	exitCode := m.Run()
	// Useful for debugging test services.
	if *debug {
		sigs := make(chan os.Signal, 1)
		done := make(chan bool, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigs
			done <- true
		}()
		log.Printf("entering debug mode: must exit (CTRL+C) and cleanup containers manually. Exit code: %d)", exitCode)
		<-done
	}
	os.Exit(exitCode)
}

// newDockerDB starts a database container and returns a usable SQL connection.
func newDockerDB(t *testing.T) (*sql.DB, error) {
	options := []testdb.OptionsFunc{
		testdb.WithBindPort(*bindPort),
		testdb.WithDebug(*debug),
	}
	var (
		db      *sql.DB
		cleanup func()
		err     error
	)
	switch *dialect {
	case dialectPostgres:
		db, cleanup, err = testdb.NewPostgres(options...)
	case dialectMySQL:
		db, cleanup, err = testdb.NewMariaDB(options...)
	default:
		return nil, fmt.Errorf("unsupported dialect: %q", *dialect)
	}
	check.NoError(t, err)
	t.Cleanup(cleanup)
	return db, nil
}
