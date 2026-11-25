package table

import (
	"log/slog"
	"time"

	"github.com/pressly/goose/v3/internal/retry"
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

func (p ProbeConfig) policy() retry.Policy {
	return retry.Policy{
		Interval:   p.IntervalDuration,
		MaxRetries: p.FailureThreshold,
		// Add +/- 25% jitter to reduce thundering herd
		JitterPercent: 25,
	}
}
