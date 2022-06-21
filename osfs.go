package goose

import (
	"io/fs"
	"os"
	"path/filepath"
)

// osFS wraps functions working with os filesystem to implement fs.FS interfaces.
type osFS struct{}

var _ fs.FS = (*osFS)(nil)

func (osFS) Open(name string) (fs.File, error) { return os.Open(filepath.FromSlash(name)) }
