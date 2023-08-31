package testdb

import (
	"context"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker/types"
)

// waitInit waits until docker container with specified id health status is healthy
func waitInit(ctx context.Context, pool *dockertest.Pool, id string) error {
	var (
		initDoneCh = make(chan struct{})
		initErr    error
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				attemptCtx, attemptCancel := context.WithTimeout(context.Background(), time.Second)
				status, err := getHealthStatus(attemptCtx, pool, id)
				attemptCancel()

				if err != nil {
					initDoneCh <- struct{}{}
					initErr = err
					return
				}

				if status == types.Healthy {
					initDoneCh <- struct{}{}
					return
				}
			}

		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-initDoneCh:
			return initErr
		}
	}
}

func getHealthStatus(ctx context.Context, pool *dockertest.Pool, id string) (string, error) {
	currentContainer, err := pool.Client.InspectContainerWithContext(id, ctx)
	if err != nil {
		return "", err
	}

	return currentContainer.State.Health.Status, nil

}
