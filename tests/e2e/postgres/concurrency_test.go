package postgres_test

import (
	"context"
	"sync"
	"testing"

	"github.com/pressly/goose/v4/internal/check"
)

func TestConcurrentProvider(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	te := newTestEnv(t, migrationsDir, nil)

	expected := 7

	ch := make(chan int64)
	var wg sync.WaitGroup
	for i := 0; i < expected; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			res, err := te.provider.UpByOne(ctx)
			if err != nil {
				t.Error(err)
				return
			}
			ch <- res.Version
		}()
	}
	go func() {
		wg.Wait()
		close(ch)
	}()
	var versions []int64
	for version := range ch {
		versions = append(versions, version)
	}
	check.Number(t, len(versions), expected)
	for i := 0; i < expected; i++ {
		check.Number(t, versions[i], int64(i+1))
	}
	version, err := te.provider.GetDBVersion(ctx)
	check.NoError(t, err)
	check.Number(t, version, expected)
}
