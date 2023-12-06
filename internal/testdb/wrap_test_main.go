package testdb

import (
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
)

const (
	// key_TESTDB_NOCLEANUP is the environment variable that disables container cleanup.
	key_TESTDB_NOCLEANUP = "TESTDB_NOCLEANUP"
	// key_TESTDB_BLOCK is the environment variable that blocks the test until a signal is received.
	key_TESTDB_BLOCK = "TESTDB_BLOCK"
)

func WrapTestMain(m *testing.M) {
	code := m.Run()
	defer func() {
		if envIsTrue(key_TESTDB_BLOCK) {
			blockUntilSignal(code)
		}
		os.Exit(code)
	}()
}

func blockUntilSignal(code int) {
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()
	fmt.Fprintf(os.Stderr, "+++ debug mode: must exit (CTRL+C) manually. (code: %d)\n", code)
	<-done
}

func envIsTrue(key string) bool {
	b, err := strconv.ParseBool(os.Getenv(key))
	return err == nil && b
}
