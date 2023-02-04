package filemetadata

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"

	"github.com/pressly/goose/v3"
)

type FileMetadata struct {
	FileType           string
	BaseName           string
	Version            int64
	Tx                 bool
	UpCount, DownCount int
}

func Run(filename string) ([]*FileMetadata, error) {
	stat, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	var files []string
	if stat.IsDir() {
		for _, pattern := range []string{"*.sql", "*.go"} {
			file, err := filepath.Glob(path.Join(filename, pattern))
			if err != nil {
				return nil, err
			}
			files = append(files, file...)
		}
	} else {
		files = append(files, filename)
	}
	sort.Strings(files)

	var metadata []*FileMetadata
	for _, f := range files {
		baseName := filepath.Base(f)
		version, err := goose.NumericComponent(f)
		if err != nil {
			return nil, fmt.Errorf("failed to parse migration file %q: %w", f, err)
		}
		switch filepath.Ext(f) {
		case ".sql":
			sqlMigration, err := parseSQLFile(f)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %q: %w", f, err)
			}
			m := convertSQLMigration(sqlMigration)
			m.Version = version
			m.BaseName = baseName
			metadata = append(metadata, m)
		case ".go":
			goMigration, err := parseGoFile(f)
			if err != nil {
				return nil, fmt.Errorf("failed to parse %q: %w", f, err)
			}
			m := convertGoMigration(goMigration)
			m.Version = version
			m.BaseName = baseName
			metadata = append(metadata, m)
		}
	}
	return metadata, nil
}
