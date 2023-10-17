package provider

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type Migration struct {
	Version       int64
	Source        string // path to .sql script or go file
	Registered    bool
	UseTx         bool
	UpFnContext   func(context.Context, *sql.Tx) error
	DownFnContext func(context.Context, *sql.Tx) error

	UpFnNoTxContext   func(context.Context, *sql.DB) error
	DownFnNoTxContext func(context.Context, *sql.DB) error
}

var registeredGoMigrations = make(map[int64]*Migration)

func SetGlobalGoMigrations(migrations []*Migration) error {
	for _, m := range migrations {
		if m == nil {
			return errors.New("cannot register nil go migration")
		}
		if _, ok := registeredGoMigrations[m.Version]; ok {
			return fmt.Errorf("go migration with version %d already registered", m.Version)
		}
		registeredGoMigrations[m.Version] = m
	}
	return nil
}

func ResetGlobalGoMigrations() {
	registeredGoMigrations = make(map[int64]*Migration)
}
