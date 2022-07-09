package goose

import (
	"io/fs"
	"math"
)

var (
	minVersion      = int64(0)
	maxVersion      = int64(math.MaxInt64)
	timestampFormat = "20060102150405"
	verbose         = false

	// base fs to lookup migrations
	baseFS fs.FS = osFS{}
)

// SetVerbose set the goose verbosity mode
func SetVerbose(v bool) {
	verbose = v
}

// SetBaseFS sets a base FS to discover migrations. It can be used with 'embed' package.
// Calling with 'nil' argument leads to default behaviour: discovering migrations from os filesystem.
// Note that modifying operations like Create will use os filesystem anyway.
func SetBaseFS(fsys fs.FS) {
	if fsys == nil {
		fsys = osFS{}
	}

	baseFS = fsys
}
