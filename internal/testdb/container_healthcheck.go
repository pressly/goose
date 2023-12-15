package testdb

import (
	"context"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker/types"
)

// containerWaitHealthy waits until docker container with specified id is healthy
func containerWaitHealthy(ctx context.Context, pool *dockertest.Pool, id string) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			attemptCtx, attemptCancel := context.WithTimeout(ctx, time.Second)
			status, err := containerHealthStatus(attemptCtx, pool, id)
			attemptCancel()
			if err != nil {
				return err
			}
			if status == types.Healthy {
				return nil
			}
		}
	}
}

func containerHealthStatus(ctx context.Context, pool *dockertest.Pool, id string) (string, error) {
	currentContainer, err := pool.Client.InspectContainerWithContext(id, ctx)
	if err != nil {
		return "", err
	}

	return currentContainer.State.Health.Status, nil

}
