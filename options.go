package retry

import (
	"math/rand"
	"time"
)

// Option defines a function type for configuring RetryConfig.
type Option func(*RetryConfig)

// WithAttempts sets the number of retry attempts.
func WithAttempts(attempts int) Option {
	return func(rc *RetryConfig) {
		rc.attempts = attempts
	}
}

// WithDelay sets the base delay for retries.
func WithDelay(delay time.Duration) Option {
	return func(rc *RetryConfig) {
		rc.baseDelay = delay
	}
}

// WithMaxDelay sets the maximum delay for retries.
func WithMaxDelay(maxDelay time.Duration) Option {
	return func(rc *RetryConfig) {
		rc.maxDelay = maxDelay
	}
}

// WithDelayType sets the delay type function for calculating delays.
func WithDelayType(delayType DelayTypeFunc) Option {
	return func(rc *RetryConfig) {
		rc.delayType = delayType
	}
}

// WithLogger sets the logger for retry operations.
func WithLogger(logger Logger) Option {
	return func(rc *RetryConfig) {
		rc.logger = logger
	}
}

// FixedDelay returns a DelayTypeFunc that uses a fixed delay.
func FixedDelay() DelayTypeFunc {
	return func(_ int, baseDelay, _ time.Duration) time.Duration {
		return baseDelay
	}
}

// ExpBackoff returns a DelayTypeFunc that uses exponential backoff.
func ExpBackoffWithJitter() DelayTypeFunc {
	return func(attempt int, baseDelay, maxDelay time.Duration) time.Duration {
		expBackoff := baseDelay * time.Duration(1<<attempt)
		jitter := time.Duration(rand.Int63n(int64(expBackoff) / 5))

		finalDelay := expBackoff + jitter
		if finalDelay > maxDelay {
			finalDelay = maxDelay
		}
		return finalDelay
	}
}
