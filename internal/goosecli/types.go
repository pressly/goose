package goosecli

import (
	"time"

	"github.com/pressly/goose/v3"
)

type migrationsStatus struct {
	Migrations []migrationStatus `json:"migrations"`
	HasPending bool              `json:"has_pending"`
}

type migrationStatus struct {
	AppliedAt string `json:"applied_at,omitempty"`
	State     string `json:"state"`
	Source    source `json:"source"`
}

func toMigrationStatus(all []*goose.MigrationStatus) migrationsStatus {
	out := migrationsStatus{
		Migrations: make([]migrationStatus, 0, len(all)),
	}
	for _, s := range all {
		var appliedAt string
		switch s.State {
		case goose.StateApplied:
			appliedAt = s.AppliedAt.Format(time.RFC3339)
		case goose.StatePending:
			out.HasPending = true
		}
		out.Migrations = append(out.Migrations, migrationStatus{
			AppliedAt: appliedAt,
			State:     string(s.State),
			Source:    toSource(s.Source),
		})
	}
	return out
}

type source struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Version int64  `json:"version"`
}

func toSource(s *goose.Source) source {
	return source{
		Type:    string(s.Type),
		Path:    s.Path,
		Version: s.Version,
	}
}

type migrationResult struct {
	Source    source `json:"source"`
	Duration  int64  `json:"duration_ms"`
	Direction string `json:"direction"`
	Empty     bool   `json:"empty"`
	Error     string `json:"error,omitempty"`
}

func toMigrationResult(results []*goose.MigrationResult) []migrationResult {
	out := make([]migrationResult, 0, len(results))
	for _, r := range results {
		var err string
		if r.Error != nil {
			err = r.Error.Error()
		}
		out = append(out, migrationResult{
			Source:    toSource(r.Source),
			Duration:  r.Duration.Milliseconds(),
			Direction: string(r.Direction),
			Empty:     r.Empty,
			Error:     err,
		})
	}
	return out
}

func truncateDuration(d time.Duration) time.Duration {
	for _, v := range []time.Duration{
		time.Second,
		time.Millisecond,
		time.Microsecond,
	} {
		if d > v {
			return d.Round(v / time.Duration(100))
		}
	}
	return d
}
