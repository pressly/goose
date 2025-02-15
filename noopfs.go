package goose

import (
	"io/fs"
	"os"
)

type noopFS struct{}

var _ fs.FS = noopFS{}

func (f noopFS) Open(_ string) (fs.File, error) {
	return nil, os.ErrNotExist
}
