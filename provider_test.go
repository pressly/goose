package goose_test

import (
	"context"
	"testing"

	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/internal/check"
)

func TestNewProvider(t *testing.T) {
	dbString := "postgresql://dbuser:password123@localhost:5432/bestofgodb?sslmode=disable"
	provider, err := goose.NewProvider("postgres", dbString, "./migrations")
	check.NoError(t, err)

	provider.Up(context.Background())
}
