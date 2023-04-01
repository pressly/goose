package goose

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type FixResult struct {
	OldPath string
	NewPath string
}

func Fix(dir string) ([]FixResult, error) {
	if dir == "" {
		return nil, fmt.Errorf("dir is required")
	}
	migrations, err := collectMigrations(osFS{}, dir, nil, false)
	if err != nil {
		return nil, err
	}
	// split into timestamped and versioned migrations
	tsMigrations, err := timestamped(migrations)
	if err != nil {
		return nil, err
	}
	vMigrations, err := versioned(migrations)
	if err != nil {
		return nil, err
	}
	// Find the next version number to use for the timestamped migrations
	// by finding the highest version number in the versioned migrations.
	var version int64 = 1
	if len(vMigrations) > 0 {
		version = vMigrations[len(vMigrations)-1].version + 1
	}
	// fix filenames by replacing timestamps with sequential versions
	results := make([]FixResult, 0, len(tsMigrations))
	for _, tsm := range tsMigrations {
		oldPath := tsm.source
		newPath := strings.Replace(
			oldPath,
			strconv.FormatInt(tsm.version, 10),
			fmt.Sprintf(seqVersionFormat, version),
			1,
		)
		if err := os.Rename(oldPath, newPath); err != nil {
			return nil, err
		}
		results = append(results, FixResult{
			OldPath: oldPath,
			NewPath: newPath,
		})
		version++
	}
	return results, nil
}

func timestamped(in []*migration) ([]*migration, error) {
	var migrations []*migration
	// assume that the user will never have more than 19700101000000 migrations
	for _, m := range in {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, fmt.Sprintf("%d", m.version))
		if err != nil {
			// probably not a timestamp
			continue
		}
		if versionTime.After(time.Unix(0, 0)) {
			migrations = append(migrations, m)
		}
	}
	return migrations, nil
}
