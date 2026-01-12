package goose

import (
	"github.com/pressly/goose/v3/database"
	"github.com/pressly/goose/v3/internal/legacystore"
)

// Dialect is the type of database dialect. It is an alias for [database.Dialect].
type Dialect = database.Dialect

const (
	DialectCustom     Dialect = database.DialectCustom
	DialectClickHouse Dialect = database.DialectClickHouse
	DialectMSSQL      Dialect = database.DialectMSSQL
	DialectMySQL      Dialect = database.DialectMySQL
	DialectPostgres   Dialect = database.DialectPostgres
	DialectRedshift   Dialect = database.DialectRedshift
	DialectSQLite3    Dialect = database.DialectSQLite3
	DialectSpanner    Dialect = database.DialectSpanner
	DialectStarrocks  Dialect = database.DialectStarrocks
	DialectTiDB       Dialect = database.DialectTiDB
	DialectTurso      Dialect = database.DialectTurso
	DialectYdB        Dialect = database.DialectYdB

	// Dialects only available to the [Provider].
	DialectAuroraDSQL Dialect = database.DialectAuroraDSQL

	// DEPRECATED: Vertica support is deprecated and will be removed in a future release.
	DialectVertica Dialect = database.DialectVertica
)

func init() {
	store, _ = legacystore.NewStore(DialectPostgres)
}

var store legacystore.Store

// SetDialect sets the dialect to use for the goose package.
func SetDialect(s string) error {
	d, err := database.ParseDialect(s)
	if err != nil {
		return err
	}
	store, err = legacystore.NewStore(d)
	return err
}
