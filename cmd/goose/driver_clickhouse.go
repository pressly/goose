//go:build !no_clickhouse
// +build !no_clickhouse

package main

import (
	_ "github.com/ClickHouse/clickhouse-go/v2"
)
