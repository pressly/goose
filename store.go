package goose

import "github.com/pressly/goose/v4/internal/dialectstore"

var store dialectstore.Store

func init() {
	store, _ = dialectstore.NewStore(DialectPostgres, DefaultTablename)
}
