package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"

	"github.com/testcontainers/testcontainers-go"
)

func init() {
	// Disable ryuk for faster tests.
	os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true")
	// Not great, but I'd like to reduce (read eliminate) the amount of logging this package does.
	testcontainers.Logger = nopLogger{}
	// Ended up filing an issue for both of these:
	// https://github.com/testcontainers/testcontainers-go/issues/1984
}

const (
	defaultLabel = "goose_test"
)

func NewClickHouse(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newClickHouse(options...)
}

func NewPostgres(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newPostgres(options...)
}

func NewMariaDB(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newMariaDB(options...)
}

func NewVertica(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newVertica(options...)
}

func NewYdb(options ...OptionsFunc) (db *sql.DB, cleanup func(), err error) {
	return newYdb(options...)
}

func maybeCleanup(p testcontainers.Container) func() {
	return func() {
		if envIsTrue(key_TESTDB_NOCLEANUP) {
			fmt.Fprintln(os.Stderr, cleanupMessage(p))
			return
		}
		if err := p.Terminate(context.Background()); err != nil {
			fmt.Fprintf(os.Stderr, "failed to terminate container: %v\n", err)
		}
	}
}

func cleanupMessage(p testcontainers.Container) string {
	msg := `+++ debug mode: skip cleanup, must manually remove container:
1. docker rm -f %s 
2. docker stop -t=1 $(docker ps -q --filter "label=goose_test")
`
	return fmt.Sprintf(msg, p.GetContainerID())
}

type nopLogger struct{}

var _ testcontainers.Logging = nopLogger{}

func (nopLogger) Printf(string, ...interface{}) {}
