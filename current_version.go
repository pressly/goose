package goose

import "context"

// GetDBVersion returns the current version of the database.
func (p *Provider) GetDBVersion(ctx context.Context) (int64, error) {
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
