package e2e

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
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
	switch *dialect {
	case dialectPostgres:
		return newDockerPostgresDB(t, *bindPort)
	case dialectMySQL:
		return newDockerMariaDB(t, *bindPort)
	}
	return nil, fmt.Errorf("unsupported dialect: %q", *dialect)
}

func newDockerPostgresDB(t *testing.T, bindPort int) (*sql.DB, error) {
	const (
		dbUsername = "postgres"
		dbPassword = "password1"
		dbHost     = "localhost"
		dbName     = "testdb"
	)
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket).
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker: %v", err)
	}
	options := &dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "14-alpine",
		Env: []string{
			"POSTGRES_USER=" + dbUsername,
			"POSTGRES_PASSWORD=" + dbPassword,
			"POSTGRES_DB=" + dbName,
			"listen_addresses = '*'",
		},
		Labels:       map[string]string{"goose_test": "1"},
		PortBindings: make(map[docker.Port][]docker.PortBinding),
	}
	if bindPort > 0 {
		options.PortBindings[docker.Port("5432/tcp")] = []docker.PortBinding{
			{HostPort: strconv.Itoa(bindPort)},
		}
	}

	container, err := pool.RunWithOptions(
		options,
		func(config *docker.HostConfig) {
			// Set AutoRemove to true so that stopped container goes away by itself.
			config.AutoRemove = true
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker container: %v", err)
	}
	t.Cleanup(func() {
		if *debug {
			// User must manually delete the Docker container.
			return
		}
		if err := pool.Purge(container); err != nil {
			log.Printf("failed to purge resource: %v", err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start resource: %v", err)
	}
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost,
		container.GetPort("5432/tcp"), // Fetch port dynamically assigned to container
		dbUsername,
		dbPassword,
		dbName,
	)
	var db *sql.DB
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet.
	if err := pool.Retry(
		func() error {
			var err error
			db, err = sql.Open(dialectPostgres, psqlInfo)
			if err != nil {
				return err
			}
			return db.Ping()
		},
	); err != nil {
		return nil, fmt.Errorf("could not connect to docker database: %v", err)
	}
	return db, nil
}

func newDockerMariaDB(t *testing.T, bindPort int) (*sql.DB, error) {
	const (
		dbUsername = "tester"
		dbPassword = "password1"
		dbHost     = "localhost"
		dbName     = "testdb"
	)
	// Uses a sensible default on windows (tcp/http) and linux/osx (socket).
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to docker: %v", err)
	}
	options := &dockertest.RunOptions{
		Repository: "mariadb",
		Tag:        "10",
		Env: []string{
			"MARIADB_USER=" + dbUsername,
			"MARIADB_PASSWORD=" + dbPassword,
			"MARIADB_ROOT_PASSWORD=" + dbPassword,
			"MARIADB_DATABASE=" + dbName,
		},
		Labels: map[string]string{"goose_test": "1"},
		// PortBindings: make(map[docker.Port][]docker.PortBinding),
	}
	if bindPort > 0 {
		options.PortBindings[docker.Port("3306/tcp")] = []docker.PortBinding{
			{HostPort: strconv.Itoa(bindPort)},
		}
	}

	container, err := pool.RunWithOptions(
		options,
		func(config *docker.HostConfig) {
			// Set AutoRemove to true so that stopped container goes away by itself.
			config.AutoRemove = true
			// config.PortBindings = options.PortBindings
			config.RestartPolicy = docker.RestartPolicy{Name: "no"}
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker container: %v", err)
	}
	t.Cleanup(func() {
		if *debug {
			// User must manually delete the Docker container.
			return
		}
		if err := pool.Purge(container); err != nil {
			log.Printf("failed to purge resource: %v", err)
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start resource: %v", err)
	}
	// MySQL DSN: username:password@protocol(address)/dbname?param=value
	dsn := fmt.Sprintf("%s:%s@(%s:%s)/%s?parseTime=true&multiStatements=true",
		dbUsername,
		dbPassword,
		dbHost,
		container.GetPort("3306/tcp"), // Fetch port dynamically assigned to container
		dbName,
	)
	var db *sql.DB
	// Exponential backoff-retry, because the application in the container
	// might not be ready to accept connections yet. Add an extra sleep
	// because mariadb containers take much longer to startup.
	time.Sleep(5 * time.Second)
	if err := pool.Retry(func() error {
		var err error
		db, err = sql.Open(dialectMySQL, dsn)
		if err != nil {
			return err
		}
		return db.Ping()
	},
	); err != nil {
		return nil, fmt.Errorf("could not connect to docker database: %v", err)
	}
	return db, nil
}
