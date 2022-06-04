package goose

import (
	"database/sql"
	"fmt"
)

// Version prints the current version of the database.
func Version(db *sql.DB, dir string, opts ...OptionsFunc) error {
	return defaultProvider.Version(db, dir, opts...)
}

// Version prints the current version of the database.
func (p *Provider) Version(db *sql.DB, dir string, opts ...OptionsFunc) error {
	option := applyOptions(opts)
	if option.noVersioning {
		var current int64
		migrations, err := p.CollectMigrations(dir, minVersion, maxVersion)
		if err != nil {
			return fmt.Errorf("failed to collect migrations: %w", err)
		}
		if len(migrations) > 0 {
			current = migrations[len(migrations)-1].Version
		}
		p.log.Printf("goose: file version %v\n", current)
		return nil
	}

	current, err := p.GetDBVersion(db)
	if err != nil {
		return err
	}
	p.log.Printf("goose: version %v\n", current)
	return nil
}

// TableName returns goose db version table name
func TableName() string {
	return defaultProvider.tableName
}

// TableName returns goose db version table name
func (p *Provider) TableName() string {
	return p.tableName
}

// SetTableName set goose db version table name
func SetTableName(n string) {
	defaultProvider.SetTableName(n)
}

// SetTableName set goose db version table name
func (p *Provider) SetTableName(n string) {
	p.tableName = n
	p.dialect.SetTableName(n)
}
