package table

import (
	"log/slog"
	"time"

	"github.com/sethvargo/go-retry"
)

// Config holds configuration for table locker.
type Config struct {
	TableName         string
	LockID            int64
	LeaseDuration     time.Duration
	HeartbeatInterval time.Duration
	LockTimeout       ProbeConfig
	UnlockTimeout     ProbeConfig

	// Optional logger for lock operations
	Logger *slog.Logger

	// Optional custom retry policy for database errors
	RetryPolicy RetryPolicyFunc
}

// ProbeConfig holds retry configuration.
type ProbeConfig struct {
	IntervalDuration time.Duration
	FailureThreshold uint64
}

// backoff returns a new backoff for one Lock or Unlock call. The backoff counts the retries it has
// used and never resets, so each call must get its own.
func (p ProbeConfig) backoff() retry.Backoff {
	// retry.NewConstant is not used here, because it panics on an interval that is not positive.
	constant := retry.BackoffFunc(func() (time.Duration, bool) {
		return p.IntervalDuration, false
	})
	return retry.WithMaxRetries(
		p.FailureThreshold,
		// Add +/- 25% jitter to reduce thundering herd
		retry.WithJitterPercent(25, constant),
	)
}
