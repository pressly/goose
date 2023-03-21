package postgres_test

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/pressly/goose/v4"
	"github.com/pressly/goose/v4/internal/check"
	"github.com/pressly/goose/v4/internal/testdb"
)

// Flags.
var (
	debug = flag.Bool(
		"debug",
		false,
		"Debug traps the test suite: useful for debugging running containers",
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
	// migrationsDir is the directory containing all migration files.
	migrationsDir = filepath.Join("testdata", "migrations")
	// seedDir is similar to migrationsDir but contains seed data
	seedDir = filepath.Join("testdata", "seed")

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
	t.Helper()
	options := []testdb.OptionsFunc{
		testdb.WithBindPort(*bindPort),
		testdb.WithDebug(*debug),
	}

	db, cleanup, err := testdb.NewPostgres(options...)
	check.NoError(t, err)
	t.Cleanup(cleanup)
	return db, nil
}

func getMaxVersionID(db *sql.DB, gooseTable string) (int64, error) {
	var gotVersion int64
	if err := db.QueryRow(
		fmt.Sprintf("select max(version_id) from %s", gooseTable),
	).Scan(&gotVersion); err != nil {
		return 0, err
	}
	return gotVersion, nil
}

func getGooseVersionCount(db *sql.DB, gooseTable string) (int64, error) {
	var gotVersion int64
	if err := db.QueryRow(
		fmt.Sprintf("SELECT count(*) FROM %s WHERE version_id > 0", gooseTable),
	).Scan(&gotVersion); err != nil {
		return 0, err
	}
	return gotVersion, nil
}

func getTableNames(db *sql.DB) ([]string, error) {
	query := `SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tableNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tableNames = append(tableNames, name)
	}
	return tableNames, nil
}

type testEnv struct {
	db       *sql.DB
	provider *goose.Provider
	opt      goose.Options
}

// newTestEnv creates a new test environment.
//
// It starts a new database container and returns a testEnv with a
// goose.Provider and a *sql.DB.
//
// If options is nil, it will use the default options. But the directory
// will always be set to the supplied dir.
func newTestEnv(t *testing.T, dir string, options *goose.Options) *testEnv {
	t.Helper()

	db, err := newDockerDB(t)
	check.NoError(t, err)

	var opt goose.Options
	if options == nil {
		opt = goose.DefaultOptions().SetVerbose(testing.Verbose())
	} else {
		opt = *options
	}
	opt = opt.SetDir(dir)

	provider, err := goose.NewProvider(goose.DialectPostgres, db, opt)
	check.NoError(t, err)
	check.NoError(t, provider.Ping(context.Background()))
	t.Cleanup(func() {
		check.NoError(t, provider.Close())
	})
	return &testEnv{
		db:       db,
		provider: provider,
		opt:      opt,
	}
}

func lastVersion(migrations []*goose.Source) int64 {
	if len(migrations) == 0 {
		return 0
	}
	return migrations[len(migrations)-1].Version
}
