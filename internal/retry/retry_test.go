package retry

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDo(t *testing.T) {
	t.Parallel()

	errTemporary := errors.New("temporary")
	errPermanent := errors.New("permanent")

	tests := []struct {
		name      string
		policy    Policy
		failures  int   // number of calls that fail before f returns nil
		failErr   error // error returned by the calls that fail
		finalErr  error // error returned after the failures; nil means success
		wantCalls int
		wantErr   error // nil means Do must succeed
	}{
		{
			name:      "success on first attempt",
			policy:    Policy{MaxRetries: 3},
			wantCalls: 1,
		},
		{
			name:      "success after retries",
			policy:    Policy{MaxRetries: 3},
			failures:  2,
			failErr:   RetryableError(errTemporary),
			wantCalls: 3,
		},
		{
			name:      "success on the last retry",
			policy:    Policy{MaxRetries: 3},
			failures:  3,
			failErr:   RetryableError(errTemporary),
			wantCalls: 4,
		},
		{
			name:      "unmarked error after retries returns at once",
			policy:    Policy{MaxRetries: 3},
			failures:  2,
			failErr:   RetryableError(errTemporary),
			finalErr:  errPermanent,
			wantCalls: 3,
			wantErr:   errPermanent,
		},
		{
			name:      "retries run out",
			policy:    Policy{MaxRetries: 3},
			failures:  10,
			failErr:   RetryableError(errTemporary),
			wantCalls: 4,
			wantErr:   errTemporary,
		},
		{
			name:      "zero retries means one attempt",
			policy:    Policy{},
			failures:  10,
			failErr:   RetryableError(errTemporary),
			wantCalls: 1,
			wantErr:   errTemporary,
		},
		{
			name:      "unmarked error returns at once",
			policy:    Policy{MaxRetries: 3},
			failures:  10,
			failErr:   errPermanent,
			wantCalls: 1,
			wantErr:   errPermanent,
		},
		{
			name:      "zero interval with jitter",
			policy:    Policy{MaxRetries: 2, JitterPercent: 25},
			failures:  10,
			failErr:   RetryableError(errTemporary),
			wantCalls: 3,
			wantErr:   errTemporary,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var calls int
			err := Do(t.Context(), tt.policy, func(context.Context) error {
				calls++
				if calls <= tt.failures {
					return tt.failErr
				}
				return tt.finalErr
			})
			require.Equal(t, tt.wantCalls, calls)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			// Do returns the marked error itself, never the mark.
			require.Same(t, tt.wantErr, err)
		})
	}
}

func TestDoWaitsIntervalBetweenAttempts(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		var offsets []time.Duration
		start := time.Now()
		err := Do(t.Context(), Policy{Interval: time.Second, MaxRetries: 3}, func(context.Context) error {
			offsets = append(offsets, time.Since(start))
			return RetryableError(errors.New("temporary"))
		})
		require.Error(t, err)
		require.Equal(t, []time.Duration{0, time.Second, 2 * time.Second, 3 * time.Second}, offsets)
		require.Equal(t, 3*time.Second, time.Since(start))
	})
}

func TestDoContextEndsDuringWait(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		errTemporary := errors.New("temporary")
		ctx, cancel := context.WithTimeout(t.Context(), 2500*time.Millisecond)
		defer cancel()

		var calls int
		start := time.Now()
		err := Do(ctx, Policy{Interval: time.Second, MaxRetries: 10}, func(context.Context) error {
			calls++
			return RetryableError(errTemporary)
		})
		require.Equal(t, 3, calls)
		require.Equal(t, 2500*time.Millisecond, time.Since(start))
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorIs(t, err, errTemporary)
	})
}

func TestDoContextEndedBeforeFirstAttempt(t *testing.T) {
	t.Parallel()
	errCause := errors.New("shutting down")
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errCause)

	var calls int
	err := Do(ctx, Policy{MaxRetries: 3}, func(context.Context) error {
		calls++
		return nil
	})
	require.Zero(t, calls)
	require.Same(t, errCause, err)
}

func TestDoNoRetryLimitEndsWithContext(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		errTemporary := errors.New("temporary")
		ctx, cancel := context.WithTimeout(t.Context(), 9750*time.Millisecond)
		defer cancel()

		var calls int
		policy := Policy{Interval: 500 * time.Millisecond, MaxRetries: Unlimited}
		err := Do(ctx, policy, func(context.Context) error {
			calls++
			return RetryableError(errTemporary)
		})
		require.Equal(t, 20, calls)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.ErrorIs(t, err, errTemporary)
	})
}

func TestDoContextCause(t *testing.T) {
	t.Parallel()
	errCause := errors.New("shutting down")
	errTemporary := errors.New("temporary")
	ctx, cancel := context.WithCancelCause(t.Context())

	err := Do(ctx, Policy{MaxRetries: 3}, func(context.Context) error {
		cancel(errCause)
		return RetryableError(errTemporary)
	})
	require.ErrorIs(t, err, errCause)
	require.ErrorIs(t, err, errTemporary)
	require.Equal(t, "shutting down: last error: temporary", err.Error())
}

func TestDoPassesContext(t *testing.T) {
	t.Parallel()
	type key struct{}
	ctx := context.WithValue(t.Context(), key{}, "value")

	err := Do(ctx, Policy{}, func(ctx context.Context) error {
		require.Equal(t, "value", ctx.Value(key{}))
		return nil
	})
	require.NoError(t, err)
}

func TestRetryableNil(t *testing.T) {
	t.Parallel()
	require.NoError(t, RetryableError(nil))
}

func TestPolicyWait(t *testing.T) {
	t.Parallel()
	const interval = 100 * time.Millisecond

	tests := []struct {
		name       string
		policy     Policy
		low, high  time.Duration
		wantSpread bool // the samples must not all be the same
	}{
		{name: "no jitter", policy: Policy{Interval: interval}, low: interval, high: interval},
		{
			name:       "25 percent",
			policy:     Policy{Interval: interval, JitterPercent: 25},
			low:        75 * time.Millisecond,
			high:       125 * time.Millisecond,
			wantSpread: true,
		},
		{
			name:       "above 100 counts as 100",
			policy:     Policy{Interval: interval, JitterPercent: 500},
			low:        0,
			high:       2 * interval,
			wantSpread: true,
		},
		{name: "zero interval", policy: Policy{JitterPercent: 25}, low: 0, high: 0},
		{name: "negative interval", policy: Policy{Interval: -time.Second, JitterPercent: 25}, low: 0, high: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			smallest, largest := tt.high, tt.low
			for range 1_000 {
				got := tt.policy.wait()
				require.GreaterOrEqual(t, got, tt.low)
				require.LessOrEqual(t, got, tt.high)
				smallest, largest = min(smallest, got), max(largest, got)
			}
			if tt.wantSpread {
				require.Less(t, smallest, largest)
			}
		})
	}
}
