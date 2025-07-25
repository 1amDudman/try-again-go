package retry

import (
	"math/rand"
	"time"
)

// Option defines a function type for configuring RetryConfig using the
// functional options pattern. This allows for flexible and extensible
// configuration of retry behavior without breaking API compatibility.
type Option func(*RetryConfig)

// WithAttempts sets the number of retry attempts for the RetryConfig.
// The attempts value determines how many times the operation will be
// retried before giving up. Must be a positive integer.
//
// Example:
//
//	retry.NewRetry(retry.WithAttempts(5)) // Will try 5 times total
func WithAttempts(attempts int) Option {
	return func(rc *RetryConfig) {
		rc.attempts = attempts
	}
}

// WithDelay sets the base delay duration between retry attempts.
// This delay is used as the foundation for delay calculations in
// both fixed and exponential backoff strategies.
//
// Example:
//
//	retry.NewRetry(retry.WithDelay(200*time.Millisecond))
func WithDelay(delay time.Duration) Option {
	return func(rc *RetryConfig) {
		rc.baseDelay = delay
	}
}

// WithMaxDelay sets the maximum delay duration that can be used between
// retry attempts. This prevents exponential backoff from growing indefinitely
// and ensures reasonable upper bounds on retry delays.
//
// Example:
//
//	retry.NewRetry(retry.WithMaxDelay(30*time.Second))
func WithMaxDelay(maxDelay time.Duration) Option {
	return func(rc *RetryConfig) {
		rc.maxDelay = maxDelay
	}
}

// WithDelayType sets the delay calculation function for retry attempts.
// This allows customization of the delay strategy (fixed, exponential, etc.).
// The function receives the attempt number, base delay, and max delay.
//
// Example:
//
//	retry.NewRetry(retry.WithDelayType(retry.ExpBackoffWithJitter()))
func WithDelayType(delayType DelayTypeFunc) Option {
	return func(rc *RetryConfig) {
		rc.delayType = delayType
	}
}

// WithLogger sets a custom logger for retry operations. The logger will
// receive detailed information about retry attempts, failures, and timing.
// Use this to integrate retry logging with your application's logging system.
//
// Example:
//
//	logger := log.New(os.Stdout, "[RETRY] ", log.LstdFlags)
//	retry.NewRetry(retry.WithLogger(logger))
func WithLogger(logger Logger) Option {
	return func(rc *RetryConfig) {
		rc.logger = logger
	}
}

// FixedDelay returns a DelayTypeFunc that uses a constant delay between
// retry attempts. The delay remains the same regardless of attempt number,
// providing predictable and consistent retry timing.
//
// This strategy is useful when you want simple, uniform delays without
// the complexity of exponential backoff.
func FixedDelay() DelayTypeFunc {
	return func(_ int, baseDelay, _ time.Duration) time.Duration {
		return baseDelay
	}
}

// ExpBackoffWithJitter returns a DelayTypeFunc that implements exponential
// backoff with random jitter. Each retry attempt doubles the delay from the
// previous attempt, with random jitter added to prevent thundering herd problems.
//
// The algorithm works as follows:
//   - Calculate exponential backoff: baseDelay * 2^attempt
//   - Add random jitter: 0 to 20% of the exponential delay
//   - Cap the result at maxDelay to prevent infinite growth
//
// This strategy is recommended for most retry scenarios as it provides
// good balance between quick recovery and system protection.
//
// Example delays with baseDelay=100ms:
//   - attempt 0: ~100-120ms
//   - attempt 1: ~200-240ms
//   - attempt 2: ~400-480ms
//   - attempt 3: limited by maxDelay
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
