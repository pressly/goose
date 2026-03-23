package dialects

import (
	"fmt"

	"github.com/pressly/goose/v3/database/dialect"
)

// NewDM returns a new [dialect.Querier] for dameng dialect.
func NewDM() dialect.QuerierExtender {
	return &dm{}
}

type dm struct{}

var _ dialect.QuerierExtender = (*dm)(nil)

func (p *dm) CreateTable(tableName string) string {
	q := `CREATE TABLE "%s" (
    "id" INTEGER PRIMARY KEY IDENTITY(1, 1),
    "version_id" BIGINT NOT NULL,
    "is_applied" BYTE NOT NULL,
    "tstamp" TIMESTAMP NOT NULL DEFAULT now()
)`
	return fmt.Sprintf(q, tableName)
}

func (p *dm) InsertVersion(tableName string) string {
	q := `INSERT INTO "%s" ("version_id", "is_applied") VALUES (?, ?)`
	return fmt.Sprintf(q, tableName)
}

func (p *dm) DeleteVersion(tableName string) string {
	q := `DELETE FROM "%s" WHERE "version_id"=?`
	return fmt.Sprintf(q, tableName)
}

func (p *dm) GetMigrationByVersion(tableName string) string {
	q := `SELECT "tstamp", "is_applied" FROM "%s" WHERE "version_id"=? ORDER BY "tstamp" DESC LIMIT 1`
	return fmt.Sprintf(q, tableName)
}

func (p *dm) ListMigrations(tableName string) string {
	q := `SELECT "version_id", "is_applied" from "%s" ORDER BY "id" DESC`
	return fmt.Sprintf(q, tableName)
}

func (p *dm) GetLatestVersion(tableName string) string {
	q := `SELECT max("version_id") FROM "%s"`
	return fmt.Sprintf(q, tableName)
}

func (p *dm) TableExists(tableName string) string {
	schemaName, tableName := parseTableIdentifier(tableName)
	if schemaName != "" {
		q := `SELECT COUNT(*) FROM "DBA_TABLES" WHERE "TABLE_NAME" = '%s' AND  "OWNER"='%s'`
		return fmt.Sprintf(q, schemaName, tableName)
	}
	q := `select count(*) from "DBA_TABLES" where "TABLE_NAME" = '%s' and  "OWNER"=(SELECT SF_GET_SCHEMA_NAME_BY_ID(CURRENT_SCHID()))`
	return fmt.Sprintf(q, tableName)
}
