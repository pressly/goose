package goose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const seqVersionTemplate = "%05v"

func Fix(dir string) error { return defaultProvider.Fix(dir) }

func (p *Provider) Fix(dir string) error {
	if p.baseDir != "" && (dir == "" || dir == ".") {
		dir = p.baseDir
	}
	// always use osFS here because it's modifying operation
	migrations, err := p.collectMigrationsFS(osFS{}, dir, minVersion, maxVersion)
	if err != nil {
		return err
	}

	// split into timestamped and versioned migrations
	tsMigrations, err := migrations.timestamped()
	if err != nil {
		return err
	}

	vMigrations, err := migrations.versioned()
	if err != nil {
		return err
	}
	// Initial version.
	version := int64(1)
	if last, err := vMigrations.Last(); err == nil {
		version = last.Version + 1
	}

	seqVerTemplate := p.seqVersionTemplate
	if seqVerTemplate == "" {
		seqVerTemplate = seqVersionTemplate
	}
	// fix filenames by replacing timestamps with sequential versions
	for _, tsm := range tsMigrations {
		oldPath := tsm.Source
		newPath := strings.Replace(
			oldPath,
			fmt.Sprintf("%d", tsm.Version),
			fmt.Sprintf(seqVerTemplate, version),
			1,
		)

		if err := os.Rename(oldPath, newPath); err != nil {
			return err
		}

		p.log.Printf("RENAMED %s => %s", filepath.Base(oldPath), filepath.Base(newPath))
		version++
	}

	return nil
}
