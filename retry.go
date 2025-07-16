package retry

import (
	"context"
	"fmt"
	"io"
	"time"
)

// DelayTypeFunc defines a function type for calculating delay.
type DelayTypeFunc func(attempt int, baseDelay, maxDelay time.Duration) time.Duration

// RetryConfig holds configuration for retry behavior.
type RetryConfig struct {
	attempts  int
	baseDelay time.Duration
	maxDelay  time.Duration
	delayType DelayTypeFunc
	logger    Logger
}

// NewRetry creates a new RetryConfig with default
// values and applies the provided options.
func NewRetry(opts ...Option) *RetryConfig {
	retry := &RetryConfig{
		attempts:  3,
		baseDelay: 100 * time.Millisecond,
		maxDelay:  1 * time.Second,
		delayType: FixedDelay(),
		logger:    nopLogger{},
	}

	for _, opt := range opts {
		opt(retry)
	}

	return retry
}

// RetryFunc is a function type that represents
// the operation to be retried.
type RetryFunc func() (io.ReadCloser, error)

// Do executes the retry logic with the provided
// context and retry function.
func (rc *RetryConfig) Do(ctx context.Context, retryFunc RetryFunc) (io.ReadCloser, error) {
	var lastErr error

	for attempt := 1; attempt <= rc.attempts; attempt++ {
		select {
		case <-ctx.Done():
			rc.logger.Printf("Retry cancelled by context on attempt %d: %v", attempt, ctx.Err())
			return nil, ctx.Err()
		default:
			data, err := retryFunc()
			if err == nil {
				return data, nil
			}

			if !isRetryable(err) {
				rc.logger.Printf("Non-retryable error on attempt %d: %v", attempt, err)
				return nil, fmt.Errorf("non-retryable error: %w", err)
			}

			if attempt != rc.attempts {
				delay := rc.baseDelay
				if rc.delayType != nil {
					delay = rc.delayType(attempt, rc.baseDelay, rc.maxDelay)
				}
				rc.logger.Printf("Attempt %d failed: %v. Retrying in %v...\n", attempt, err, delay)
				time.Sleep(delay)
			} else {
				lastErr = err
			}
		}
	}

	rc.logger.Printf("All %d attempts failed. Last error: %v", rc.attempts, lastErr)
	return nil, fmt.Errorf("all attempts failed, the last error: %w", lastErr)
}
