package retry

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"time"
)

const Unlimited uint64 = math.MaxUint64

type Policy struct {
	Interval      time.Duration
	MaxRetries    uint64
	JitterPercent uint64
}

func RetryableError(err error) error {
	if err == nil {
		return nil
	}
	return &temporaryError{cause: err}
}

func Do(ctx context.Context, policy Policy, f func(context.Context) error) error {
	if err := context.Cause(ctx); err != nil {
		return err
	}
	remaining := policy.MaxRetries
	for {
		err := f(ctx)
		if err == nil {
			return nil
		}
		temporary, ok := errors.AsType[*temporaryError](err)
		if !ok {
			return err
		}
		if remaining == 0 {
			return temporary.cause
		}
		remaining--
		if ctxErr := sleep(ctx, policy.wait()); ctxErr != nil {
			return fmt.Errorf("%w: last error: %w", ctxErr, temporary.cause)
		}
	}
}

func (p Policy) wait() time.Duration {
	if p.Interval <= 0 || p.JitterPercent == 0 {
		return max(p.Interval, 0)
	}
	spread := float64(p.Interval) * float64(min(p.JitterPercent, 100)) / 100
	offset := (rand.Float64()*2 - 1) * spread
	return p.Interval + time.Duration(offset)
}

type temporaryError struct {
	cause error
}

func (e *temporaryError) Error() string { return e.cause.Error() }

func (e *temporaryError) Unwrap() error { return e.cause }

func sleep(ctx context.Context, d time.Duration) error {
	if d > 0 {
		timer := time.NewTimer(d)
		defer timer.Stop()
		select {
		case <-ctx.Done():
		case <-timer.C:
		}
	}
	return context.Cause(ctx)
}
