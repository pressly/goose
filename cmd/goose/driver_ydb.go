//go:build !no_ydb

package main

import (
	"strings"

	"github.com/ydb-platform/ydb-go-sdk/v3"
	"github.com/ydb-platform/ydb-go-sdk/v3/config"
)

func init() {
	ydb.RegisterDsnParser(func(dsn string) (opts []ydb.Option, _ error) {
		var v = version
		if v == "" {
			v = versionFromBuildInfo()
		}

		return []ydb.Option{
			ydb.With(config.WithBuildInfo("goose", strings.TrimSpace(v))),
		}, nil
	})
}
