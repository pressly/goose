package goose

import (
	"context"
	"database/sql"
)

type Connection interface {
	Close() error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Exec(query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (Rows, error)
	Query(query string, args ...any) (Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) Row
	QueryRow(query string, args ...any) Row
	Begin() (Tx, error)
}

type Tx interface {
	Commit() error
	Rollback() error
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type Rows interface {
	Next() bool
	Err() error
	Scan(dest ...any) error
	Close() error
}

type Row interface {
	Scan(dest ...interface{}) error
}

type SqlDbToGooseAdapter struct {
	Conn *sql.DB
}

func (p SqlDbToGooseAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.Conn.Exec(query, args...)
}

func (p SqlDbToGooseAdapter) Exec(query string, args ...any) (sql.Result, error) {
	return p.ExecContext(context.Background(), query, args...)
}

func (p SqlDbToGooseAdapter) Query(query string, args ...any) (Rows, error) {
	return p.QueryContext(context.Background(), query, args...)
}

func (p SqlDbToGooseAdapter) QueryContext(ctx context.Context, query string, args ...any) (Rows, error) {
	return p.Conn.Query(query, args...)
}

func (p SqlDbToGooseAdapter) QueryRow(query string, args ...any) Row {
	return p.QueryRowContext(context.Background(), query, args...)
}

func (p SqlDbToGooseAdapter) QueryRowContext(ctx context.Context, query string, args ...any) Row {
	return p.Conn.QueryRow(query, args...)
}

func (p SqlDbToGooseAdapter) Close() error {
	return p.Conn.Close()
}

func (p SqlDbToGooseAdapter) Begin() (Tx, error) {
	return p.Conn.Begin()
}
