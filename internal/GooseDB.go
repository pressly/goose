package internal

import (
	"context"
	"database/sql"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
)

type GooseDB interface {
	Close() error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	Exec(query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (GooseRows, error)
	Query(query string, args ...any) (GooseRows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) GooseRow
	QueryRow(query string, args ...any) GooseRow
	Begin() (GooseTx, error)
}

type GooseTx interface {
	Commit() error
	Rollback() error
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type GooseRows interface {
	Next() bool
	Err() error
	Scan(dest ...any) error
	Close() error
}

type GooseRow interface {
	Scan(dest ...interface{}) error
}

type PgxToGooseAdapter struct {
	Conn *pgx.Conn
}

func (p PgxToGooseAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	t, err := p.Conn.Exec(ctx, query, args...)
	return PgxToSqlResult{ct: t}, err

}

func (p PgxToGooseAdapter) Exec(query string, args ...any) (sql.Result, error) {
	return p.ExecContext(context.Background(), query, args...)
}

func (p PgxToGooseAdapter) Query(query string, args ...any) (GooseRows, error) {
	return p.QueryContext(context.Background(), query, args...)
}

func (p PgxToGooseAdapter) QueryContext(ctx context.Context, query string, args ...any) (GooseRows, error) {
	rows, err := p.Conn.Query(ctx, query, args...)
	return PgxToGooseRows{rows: rows}, err
}

func (p PgxToGooseAdapter) QueryRow(query string, args ...any) GooseRow {
	return p.QueryRowContext(context.Background(), query, args...)
}

func (p PgxToGooseAdapter) QueryRowContext(ctx context.Context, query string, args ...any) GooseRow {
	return p.Conn.QueryRow(ctx, query, args...)
}

func (p PgxToGooseAdapter) Close() error {
	return p.Conn.Close(context.Background())
}

func (p PgxToGooseAdapter) Begin() (GooseTx, error) {
	tx, err := p.Conn.Begin(context.Background())
	return PgxToGooseTx{tx}, err
}

type PgxToGooseTx struct {
	tx pgx.Tx
}

func (p PgxToGooseTx) Commit() error {
	return p.tx.Commit(context.Background())
}

func (p PgxToGooseTx) Rollback() error {
	return p.tx.Rollback(context.Background())
}

func (p PgxToGooseTx) Exec(query string, args ...interface{}) (sql.Result, error) {
	t, err := p.tx.Exec(context.Background(), query, args...)
	return PgxToSqlResult{ct: t}, err
}

type PgxToSqlResult struct {
	ct pgconn.CommandTag
}

func (p PgxToSqlResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (p PgxToSqlResult) RowsAffected() (int64, error) {
	return p.ct.RowsAffected(), nil
}

type PgxToGooseRows struct {
	rows pgx.Rows
}

func (p PgxToGooseRows) Next() bool {
	return p.rows.Next()
}

func (p PgxToGooseRows) Err() error {
	return p.rows.Err()
}

func (p PgxToGooseRows) Scan(dest ...any) error {
	return p.rows.Scan(dest...)
}

func (p PgxToGooseRows) Close() error {
	p.rows.Close()
	return nil
}

type SqlToGooseAdapter struct {
	Conn *sql.DB
}

func (p SqlToGooseAdapter) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return p.Conn.Exec(query, args...)
}

func (p SqlToGooseAdapter) Exec(query string, args ...any) (sql.Result, error) {
	return p.ExecContext(context.Background(), query, args...)
}

func (p SqlToGooseAdapter) Query(query string, args ...any) (GooseRows, error) {
	return p.QueryContext(context.Background(), query, args...)
}

func (p SqlToGooseAdapter) QueryContext(ctx context.Context, query string, args ...any) (GooseRows, error) {
	return p.Conn.Query(query, args...)
}

func (p SqlToGooseAdapter) QueryRow(query string, args ...any) GooseRow {
	return p.QueryRowContext(context.Background(), query, args...)
}

func (p SqlToGooseAdapter) QueryRowContext(ctx context.Context, query string, args ...any) GooseRow {
	return p.Conn.QueryRow(query, args...)
}

func (p SqlToGooseAdapter) Close() error {
	return p.Conn.Close()
}

func (p SqlToGooseAdapter) Begin() (GooseTx, error) {
	return p.Conn.Begin()
}
