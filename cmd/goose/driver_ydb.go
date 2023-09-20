//go:build !no_ydb
// +build !no_ydb

package main

import (
	_ "github.com/ydb-platform/ydb-go-sdk/v3"
)
