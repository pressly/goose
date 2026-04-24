//go:build !no_ydb

package main

import (
	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/config"
)

func init() {
	ydb.RegisterDsnParser(func(dsn string) (opts []ydb.Option, _ error) {
		return []ydb.Option{
			ydb.With(config.WithBuildInfo("goose", version)),
		}, nil
	})
}
