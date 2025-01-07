package migrationstatsos

import (
	"io"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v4/internal/migrationstats"
)

// NewFileWalker returns a new FileWalker for the given filenames.
//
// Filenames without a .sql or .go extension are ignored.
func NewFileWalker(filenames ...string) migrationstats.FileWalker {
	return &fileWalker{
		filenames: filenames,
	}
}

type fileWalker struct {
	filenames []string
}

var _ migrationstats.FileWalker = (*fileWalker)(nil)

func (f *fileWalker) Walk(fn func(filename string, r io.Reader) error) error {
	for _, filename := range f.filenames {
		ext := filepath.Ext(filename)
		if ext != ".sql" && ext != ".go" {
			continue
		}
		if err := walk(filename, fn); err != nil {
			return err
		}
	}
	return nil
}

func walk(filename string, fn func(filename string, r io.Reader) error) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	return fn(filename, file)
}
