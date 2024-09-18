package cli

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

func convertMigrationStatus(all []*goose.MigrationStatus) migrationsStatus {
	out := migrationsStatus{
		Migrations: make([]migrationStatus, 0, len(all)),
	}
	for _, s := range all {
		var appliedAt string
		switch s.State {
		case goose.StateApplied:
			appliedAt = s.AppliedAt.Format(time.DateTime)
		case goose.StatePending:
			out.HasPending = true
		}
		out.Migrations = append(out.Migrations, migrationStatus{
			AppliedAt: appliedAt,
			State:     string(s.State),
			Source:    convertSource(s.Source),
		})
	}
	return out
}

type source struct {
	Type    string `json:"type"`
	Path    string `json:"path"`
	Version int64  `json:"version"`
}

func convertSource(s *goose.Source) source {
	return source{
		Type:    string(s.Type),
		Path:    s.Path,
		Version: s.Version,
	}
}
