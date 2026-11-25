package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	resultSuccess = iota
	resultRetryable
	resultPermanent
	resultKinds
)

func FuzzDo(f *testing.F) {
	f.Add(uint8(3), uint16(1000), uint8(0), []byte{1, 1, 1, 1, 1}, uint8(255))
	f.Add(uint8(3), uint16(1000), uint8(25), []byte{1, 1, 0}, uint8(255))
	f.Add(uint8(0), uint16(0), uint8(0), []byte{1}, uint8(255))
	f.Add(uint8(5), uint16(10), uint8(100), []byte{1, 2}, uint8(255))
	f.Add(uint8(5), uint16(10), uint8(50), []byte{1, 1, 1}, uint8(1))
	f.Add(uint8(5), uint16(10), uint8(50), []byte{0}, uint8(0))
	f.Add(uint8(5), uint16(10), uint8(50), []byte{2}, uint8(0))

	f.Fuzz(func(t *testing.T, maxRetries uint8, intervalMillis uint16, jitterPercent uint8, results []byte, cancelAt uint8) {
		policy := Policy{
			Interval:      time.Duration(intervalMillis) * time.Millisecond,
			MaxRetries:    uint64(maxRetries),
			JitterPercent: uint64(jitterPercent),
		}
		// resultAt returns the kind of result for each call. Calls past the end of results succeed.
		resultAt := func(call int) int {
			if call >= len(results) {
				return resultSuccess
			}
			return int(results[call] % resultKinds)
		}
		errAt := func(call int) error {
			return fmt.Errorf("error from call %d", call)
		}
		errCause := errors.New("canceled by test")

		synctest.Test(t, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(t.Context())
			defer cancel(nil)

			var calls int
			var returned []error
			start := time.Now()
			err := Do(ctx, policy, func(context.Context) error {
				call := calls
				calls++
				if call == int(cancelAt) {
					cancel(errCause)
				}
				var callErr error
				switch resultAt(call) {
				case resultRetryable:
					callErr = errAt(call)
					returned = append(returned, callErr)
					return RetryableError(callErr)
				case resultPermanent:
					callErr = errAt(call)
					returned = append(returned, callErr)
					return callErr
				}
				returned = append(returned, nil)
				return nil
			})
			elapsed := time.Since(start)

			// Model: walk the results the same way Do should.
			wantCalls := 0
			for call := 0; ; call++ {
				wantCalls++
				if resultAt(call) != resultRetryable {
					break
				}
				if uint64(call) == policy.MaxRetries {
					break
				}
				if call >= int(cancelAt) {
					break
				}
			}
			require.Equal(t, wantCalls, calls)

			last := calls - 1
			lastErr := returned[last]
			retriesLeft := uint64(last) < policy.MaxRetries
			canceled := last >= int(cancelAt)
			switch resultAt(last) {
			case resultSuccess:
				require.NoError(t, err)
			case resultPermanent:
				require.Same(t, lastErr, err)
			case resultRetryable:
				if !retriesLeft {
					require.Same(t, lastErr, err)
					break
				}
				require.True(t, canceled)
				require.ErrorIs(t, err, errCause)
				require.ErrorIs(t, err, lastErr)
			}

			// Do waits once between each pair of calls. A canceled context ends the wait at once,
			// and the context is only canceled inside f, so every wait that ran was a full wait.
			waits := time.Duration(calls - 1)
			spread := time.Duration(float64(policy.Interval) * float64(min(policy.JitterPercent, 100)) / 100)
			slack := waits // time.Duration truncation can lose up to 1ns per wait
			require.GreaterOrEqual(t, elapsed, waits*(policy.Interval-spread)-slack)
			require.LessOrEqual(t, elapsed, waits*(policy.Interval+spread)+slack)
		})
	})
}

func FuzzPolicyWait(f *testing.F) {
	f.Add(int64(time.Second), uint8(25))
	f.Add(int64(0), uint8(25))
	f.Add(int64(-time.Second), uint8(25))
	f.Add(int64(time.Millisecond), uint8(100))
	f.Add(int64(1), uint8(100))

	f.Fuzz(func(t *testing.T, interval int64, percent uint8) {
		// Keep intervals in a range a real caller can use: up to one day either way.
		const day = int64(24 * time.Hour)
		policy := Policy{
			Interval:      time.Duration(interval % day),
			JitterPercent: uint64(percent),
		}
		got := policy.wait()
		require.GreaterOrEqual(t, got, time.Duration(0))
		if policy.Interval <= 0 {
			require.Zero(t, got)
			return
		}
		spread := time.Duration(float64(policy.Interval) * float64(min(policy.JitterPercent, 100)) / 100)
		require.GreaterOrEqual(t, got, policy.Interval-spread-1)
		require.LessOrEqual(t, got, policy.Interval+spread+1)
	})
}
