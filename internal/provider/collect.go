package provider

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pressly/goose/v3"
)

// fileSources represents a collection of migration files on the filesystem.
type fileSources struct {
	sqlSources []Source
	goSources  []Source
}

// TODO(mf): remove?
func (s *fileSources) lookup(t MigrationType, version int64) *Source {
	switch t {
	case TypeGo:
		for _, source := range s.goSources {
			if source.Version == version {
				return &source
			}
		}
	case TypeSQL:
		for _, source := range s.sqlSources {
			if source.Version == version {
				return &source
			}
		}
	}
	return nil
}

// collectFilesystemSources scans the file system for migration files that have a numeric prefix
// (greater than one) followed by an underscore and a file extension of either .go or .sql. fsys may
// be nil, in which case an empty fileSources is returned.
//
// If strict is true, then any error parsing the numeric component of the filename will result in an
// error. The file is skipped otherwise.
//
// This function DOES NOT parse SQL migrations or merge registered Go migrations. It only collects
// migration sources from the filesystem.
func collectFilesystemSources(fsys fs.FS, strict bool, excludes map[string]bool) (*fileSources, error) {
	if fsys == nil {
		return new(fileSources), nil
	}
	sources := new(fileSources)
	versionToBaseLookup := make(map[int64]string) // map[version]filepath.Base(fullpath)
	for _, pattern := range []string{
		"*.sql",
		"*.go",
	} {
		files, err := fs.Glob(fsys, pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob pattern %q: %w", pattern, err)
		}
		for _, fullpath := range files {
			base := filepath.Base(fullpath)
			// Skip explicit excludes or Go test files.
			if excludes[base] || strings.HasSuffix(base, "_test.go") {
				continue
			}
			// If the filename has a valid looking version of the form: NUMBER_.{sql,go}, then use
			// that as the version. Otherwise, ignore it. This allows users to have arbitrary
			// filenames, but still have versioned migrations within the same directory. For
			// example, a user could have a helpers.go file which contains unexported helper
			// functions for migrations.
			version, err := goose.NumericComponent(base)
			if err != nil {
				if strict {
					return nil, fmt.Errorf("failed to parse numeric component from %q: %w", base, err)
				}
				continue
			}
			// Ensure there are no duplicate versions.
			if existing, ok := versionToBaseLookup[version]; ok {
				return nil, fmt.Errorf("found duplicate migration version %d:\n\texisting:%v\n\tcurrent:%v",
					version,
					existing,
					base,
				)
			}
			switch filepath.Ext(base) {
			case ".sql":
				sources.sqlSources = append(sources.sqlSources, Source{
					Type:    TypeSQL,
					Path:    fullpath,
					Version: version,
				})
			case ".go":
				sources.goSources = append(sources.goSources, Source{
					Type:    TypeGo,
					Path:    fullpath,
					Version: version,
				})
			default:
				// Should never happen since we already filtered out all other file types.
				return nil, fmt.Errorf("unknown migration type: %s", base)
			}
			// Add the version to the lookup map.
			versionToBaseLookup[version] = base
		}
	}
	return sources, nil
}

func merge(sources *fileSources, registerd map[int64]*goMigration) ([]*migration, error) {
	var migrations []*migration
	migrationLookup := make(map[int64]*migration)
	// Add all SQL migrations to the list of migrations.
	for _, source := range sources.sqlSources {
		m := &migration{
			Source: source,
			SQL:    nil, // SQL migrations are parsed lazily.
		}
		migrations = append(migrations, m)
		migrationLookup[source.Version] = m
	}
	// If there are no Go files in the filesystem and no registered Go migrations, return early.
	if len(sources.goSources) == 0 && len(registerd) == 0 {
		return migrations, nil
	}
	// Return an error if the given sources contain a versioned Go migration that has not been
	// registered. This is a sanity check to ensure users didn't accidentally create a valid looking
	// Go migration file on disk and forget to register it.
	//
	// This is almost always a user error.
	var unregistered []string
	for _, s := range sources.goSources {
		if _, ok := registerd[s.Version]; !ok {
			unregistered = append(unregistered, s.Path)
		}
	}
	if len(unregistered) > 0 {
		return nil, unregisteredError(unregistered)
	}
	// Add all registered Go migrations to the list of migrations, checking for duplicate versions.
	//
	// Important, users can register Go migrations manually via goose.Add_ functions. These
	// migrations may not have a corresponding file on disk. Which is fine! We include them
	// wholesale as part of migrations. This allows users to build a custom binary that only embeds
	// the SQL migration files.
	for version, r := range registerd {
		fullpath := r.fullpath
		if fullpath == "" {
			if s := sources.lookup(TypeGo, version); s != nil {
				fullpath = s.Path
			}
		}
		// Ensure there are no duplicate versions.
		if existing, ok := migrationLookup[version]; ok {
			fullpath := r.fullpath
			if fullpath == "" {
				fullpath = "manually registered (no source)"
			}
			return nil, fmt.Errorf("found duplicate migration version %d:\n\texisting:%v\n\tcurrent:%v",
				version,
				existing.Source.Path,
				fullpath,
			)
		}
		m := &migration{
			Source: Source{
				Type:    TypeGo,
				Path:    fullpath, // May be empty if migration was registered manually.
				Version: version,
			},
			Go: r,
		}
		migrations = append(migrations, m)
		migrationLookup[version] = m
	}
	// Sort migrations by version in ascending order.
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Source.Version < migrations[j].Source.Version
	})
	return migrations, nil
}

func unregisteredError(unregistered []string) error {
	const (
		hintURL = "https://github.com/pressly/goose/tree/master/examples/go-migrations"
	)
	f := "file"
	if len(unregistered) > 1 {
		f += "s"
	}
	var b strings.Builder

	b.WriteString(fmt.Sprintf("error: detected %d unregistered Go %s:\n", len(unregistered), f))
	for _, name := range unregistered {
		b.WriteString("\t" + name + "\n")
	}
	hint := fmt.Sprintf("hint: go functions must be registered and built into a custom binary see:\n%s", hintURL)
	b.WriteString(hint)
	b.WriteString("\n")

	return errors.New(b.String())
}

type noopFS struct{}

var _ fs.FS = noopFS{}

func (f noopFS) Open(name string) (fs.File, error) {
	return nil, os.ErrNotExist
}
