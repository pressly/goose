package main

import (
	"database/sql"
	"embed"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
)

type Migration struct {
	db *sql.DB
}

//go:embed migrations/*.sql
var embedMigrations embed.FS

func NewMigration(pool *pgxpool.Pool) (*Migration, error) {
	if pool == nil {
		return &Migration{}, errors.New("pool is nil")
	}

	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return &Migration{}, err
	}

	cp := pool.Config().ConnConfig.ConnString()
	db, err := sql.Open("pgx/v5", cp)
	if err != nil {
		return &Migration{}, err
	}

	return &Migration{db: db}, nil
}

func (m *Migration) Up() error {
	if err := goose.Up(m.db, "migrations"); err != nil {
		return err
	}
	return nil
}

func (m *Migration) Down() error {
	if err := goose.Down(m.db, "migrations"); err != nil {
		return err
	}
	return nil
}
