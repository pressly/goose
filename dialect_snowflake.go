package goose

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/snowflakedb/gosnowflake"
)

// SnowflakeDialect contains interface methods for performing migrations on snowflake.
type SnowflakeDialect struct{}

const (
	createSnowflakeVersionTablePKSequenceSQLTpl = `
		create sequence if not exists  %s
			start = 1
			increment = 1
			comment = 'sequence used as primary key for the goose version table primary key';
		`

	createSnowflakeVersionTableSQLTpl = `
		create table if not exists %s (
			id number primary key default %s.nextval,
			version_id bigint not null,
			is_applied boolean not null,
			tstamp timestamp default current_timestamp
		);
	`

	dropSnowflakeVersionTablePKSequenceSQLTpl = `drop sequence if exists %s;`

	dropSnowflakeVersionTableSQLTpl = `drop table if exists %s;`
)

func (s SnowflakeDialect) createVersionTableSQL() string {
	createSeqSQL := fmt.Sprintf(createSnowflakeVersionTablePKSequenceSQLTpl, snowflakePKSequenceName())
	createTableSQL := fmt.Sprintf(createSnowflakeVersionTableSQLTpl, TableName(), snowflakePKSequenceName())
	return createSeqSQL + createTableSQL
}

// Snowflake automatically commits transactions for each ddl statement so rollbacks need to be performed manually
func (s SnowflakeDialect) rollbackCreateVersionTable(txn *sql.Tx) error {
	dropSeqSQL := fmt.Sprintf(dropSnowflakeVersionTablePKSequenceSQLTpl, snowflakePKSequenceName())
	dropTableSQL := fmt.Sprintf(dropSnowflakeVersionTableSQLTpl, TableName())
	_, err := txn.ExecContext(s.getContext(context.Background()), dropTableSQL+dropSeqSQL)
	return err
}

func (s SnowflakeDialect) insertVersionSQL() string {
	return fmt.Sprintf("insert into %s (version_id, is_applied) values (?, ?);", TableName())
}

func (s SnowflakeDialect) dbVersionQuery(db *sql.DB) (*sql.Rows, error) {
	rows, err := db.Query(fmt.Sprintf("select version_id, is_applied from %s order by id desc", TableName()))
	if err != nil {
		return nil, err
	}

	return rows, err
}

func (s SnowflakeDialect) migrationSQL() string {
	return fmt.Sprintf("select tstamp, is_applied from %s where version_id=? order by tstamp desc limit 1", TableName())
}

func (s SnowflakeDialect) deleteVersionSQL() string {
	return fmt.Sprintf("delete from %s where version_id=?;", TableName())
}

func (s SnowflakeDialect) getContext(baseContext context.Context) context.Context {
	ctx, _ := gosnowflake.WithMultiStatement(baseContext, 2)
	return ctx
}

func snowflakePKSequenceName() string {
	return TableName() + "_pk_seq"
}
