package goose

import (
    "database/sql"
    "fmt"
)

type duckdbDialect struct{}

func init() {
    RegisterDialect("duckdb", duckdbDialect{})
}

// Create the migrations table if it doesn't exist
func (d duckdbDialect) CreateVersionTableSQL() string {
    return `
        CREATE TABLE IF NOT EXISTS goose_db_version (
            id INTEGER PRIMARY KEY,
            version_id BIGINT NOT NULL,
            is_applied BOOL NOT NULL,
            tstamp TIMESTAMP DEFAULT (CURRENT_TIMESTAMP)
        );
    `
}

func (d duckdbDialect) InsertVersionSQL() string {
    return `INSERT INTO goose_db_version (version_id, is_applied) VALUES (?, ?);`
}

func (d duckdbDialect) DeleteVersionSQL() string {
    return `DELETE FROM goose_db_version WHERE version_id = ?;`
}

func (d duckdbDialect) DBVersion(db *sql.DB) (int64, error) {
    query := `SELECT version_id FROM goose_db_version WHERE is_applied = true ORDER BY id DESC LIMIT 1;`
    var version int64
    err := db.QueryRow(query).Scan(&version)
    if err == sql.ErrNoRows {
        return 0, nil
    }
    return version, err
}

func (d duckdbDialect) MigrationExistsSQL() string {
    return `SELECT COUNT(1) FROM goose_db_version WHERE version_id = ? AND is_applied = true;`
}

