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
	sources, err := collect(osFS{}, dir, true, nil)
	if err != nil {
		return nil, err
	}
	// split into timestamped and versioned sources
	tsSources, err := timestamped(sources)
	if err != nil {
		return nil, err
	}
	vSources, err := versioned(sources)
	if err != nil {
		return nil, err
	}
	// Find the next version number to use for the timestamped migrations
	// by finding the highest version number in the versioned migrations.
	var version int64 = 1
	if len(vSources) > 0 {
		version = vSources[len(vSources)-1] + 1
	}
	// fix filenames by replacing timestamps with sequential versions
	results := make([]FixResult, 0, len(tsSources))
	for _, tsm := range tsSources {
		oldPath := tsm.Fullpath
		newPath := strings.Replace(
			oldPath,
			strconv.FormatInt(tsm.Version, 10),
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

func timestamped(in []*Source) ([]*Source, error) {
	var out []*Source
	// assume that the user will never have more than 19700101000000 migrations
	for _, src := range in {
		// parse version as timestamp
		versionTime, err := time.Parse(timestampFormat, fmt.Sprintf("%d", src.Version))
		if err != nil {
			// probably not a timestamp
			continue
		}
		if versionTime.After(time.Unix(0, 0)) {
			out = append(out, src)
		}
	}
	return out, nil
}
