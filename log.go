package goose

import (
	std "log"
	"os"
)

var log Logger = std.New(os.Stderr, "", std.LstdFlags)

// Logger is standart logger interface
type Logger interface {
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
	Print(v ...interface{})
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

// SetLogger sets the logger for package output
func SetLogger(l Logger) {
	log = l
}
