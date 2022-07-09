package goose

import "context"

// CurrentVersion prints the current version of the database.
func (p *Provider) CurrentVersion(ctx context.Context) (int64, error) {
	var migrationRow migrationRow
	err := p.db.QueryRowContext(
		ctx,
		p.dialect.GetLatestMigration(),
	).Scan(
		&migrationRow.ID,
		&migrationRow.VersionID,
		&migrationRow.Timestamp,
	)
	if err != nil {
		return 0, err
	}
	return migrationRow.VersionID, nil
}
