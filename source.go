package goose

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type SourceType int

const (
	SourceTypeGo SourceType = iota + 1
	SourceTypeSQL
)

func (t SourceType) String() string {
	switch t {
	case SourceTypeGo:
		return "go"
	case SourceTypeSQL:
		return "sql"
	default:
		return "unknown"
	}
}

// Source represents a single migration source file on disk.
type Source struct {
	// Full path to the migration file.
	//
	// Example: /path/to/migrations/001_create_users_table.sql
	Fullpath string
	// Version is the version of the migration.
	Version int64
	// Type is the type of migration.
	Type SourceType
}

// Collect returns a slice of Sources found in the given directory.
func Collect(dir string, excludes []string) ([]*Source, error) {
	return collect(osFS{}, dir, false, excludes)
}

func collect(fsys fs.FS, dir string, strict bool, excludes []string) ([]*Source, error) {
	if _, err := fs.Stat(fsys, dir); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("directory does not exist: %s", dir)
	}
	excludeLookup := make(map[string]bool, len(excludes))
	for _, v := range excludes {
		excludeLookup[v] = true
	}
	// Lookup map to ensure there are no duplicate versions.
	versionToBase := make(map[int64]string)

	var sources []*Source
	for _, pattern := range []string{"*.sql", "*.go"} {
		files, err := fs.Glob(fsys, path.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		for _, name := range files {
			base := filepath.Base(name)
			// Skip explicit excludes or Go test files.
			if excludeLookup[base] || strings.HasSuffix(base, "_test.go") {
				continue
			}
			// If the filename has a valid looking version of the form: 001_, then use that as the
			// version. Otherwise, ignore it. This allows users to have arbitrary filenames, but
			// still have versioned migrations. For example, a user could have a helpers.go file
			// which contains unexported helper functions for migrations, and it would not be
			// considered a migration.
			version, err := NumericComponent(base)
			if err != nil {
				if strict {
					return nil, err
				}
				continue
			}
			if version < 1 {
				return nil, fmt.Errorf("invalid version number %d: %s", version, base)
			}
			if existing, ok := versionToBase[version]; ok {
				return nil, fmt.Errorf("found duplicate migration version %d:\n\texisting:%v\n\tcurrent:%v",
					version,
					existing,
					base,
				)
			}
			src := &Source{
				Fullpath: name,
				Version:  version,
			}
			switch filepath.Ext(base) {
			case ".sql":
				src.Type = SourceTypeSQL
			case ".go":
				src.Type = SourceTypeSQL
			}
			// Append to the list of sources and update the lookup map.
			sources = append(sources, src)
			versionToBase[version] = base
		}
	}
	// Sort in ascending order by version id
	sort.Slice(sources, func(i, j int) bool {
		return sources[i].Version < sources[j].Version
	})
	return sources, nil
}

// NumericComponent parses the version from the migration file name.
//
// XXX_descriptivename.ext where XXX specifies the version number and ext specifies the type of
// migration, either .sql or .go.
func NumericComponent(filename string) (int64, error) {
	base := filepath.Base(filename)
	if ext := filepath.Ext(base); ext != ".go" && ext != ".sql" {
		return 0, errors.New("migration file does not have .sql or .go file extension")
	}
	idx := strings.Index(base, "_")
	if idx < 0 {
		return 0, errors.New("no filename separator '_' found")
	}
	n, err := strconv.ParseInt(base[:idx], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse version: %w", err)
	}
	if n < 1 {
		return 0, errors.New("migration version must be greater than zero")
	}
	return n, nil
}
