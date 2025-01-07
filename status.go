package goose

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/pressly/goose/v4/internal/migration"
)

type MigrationStatus struct {
	Applied   bool
	AppliedAt time.Time
	Source    *Source
}

type StatusOptions struct {
	// IncludeNilMigrations will include a migration status for a record in the database but not in
	// the filesystem. This is common when migration files are squashed and replaced with a single
	// migration file referencing a version that has already been applied, such as
	// 00001_squashed.go.
	// includeNilMigrations bool

	// Limit limits the number of migrations returned. Default is 0, which means no limit.
	// limit int
	// Sort order possible values are: ASC and DESC. Default is ASC.
	// order string
}

// Status returns the status of all migrations. The returned slice is ordered by ascending version.
//
// The provided StatusOptions is optional and may be nil if defaults should be used.
//
// It is safe for concurrent use.
func (p *Provider) Status(ctx context.Context, opts *StatusOptions) (_ []*MigrationStatus, retErr error) {
	conn, cleanup, err := p.initialize(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		retErr = errors.Join(retErr, cleanup())
	}()

	// TODO(mf): add support for limit and order. Also would be nice to refactor the list query to
	// support limiting the set.

	status := make([]*MigrationStatus, 0, len(p.migrations))
	for _, m := range p.migrations {
		migrationStatus := &MigrationStatus{
			Source: convertMigration(m),
		}
		dbResult, err := p.store.GetMigration(ctx, conn, m.Version)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		if dbResult != nil {
			migrationStatus.Applied = true
			migrationStatus.AppliedAt = dbResult.Timestamp
		}
		status = append(status, migrationStatus)
	}

	return status, nil
}

func convertMigration(m *migration.Migration) *Source {
	var typ SourceType
	switch {
	case m.IsGo():
		typ = SourceTypeGo
	case m.IsSQL():
		typ = SourceTypeSQL
	}
	return &Source{
		Fullpath: m.Fullpath,
		Version:  m.Version,
		Type:     typ,
	}
}
