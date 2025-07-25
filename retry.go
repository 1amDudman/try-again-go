package retry

import (
	"context"
	"fmt"
	"io"
	"time"
)

// DelayTypeFunc defines a function type for calculating retry delays.
// Implementations receive the current attempt number (0-based), base delay,
// and maximum delay, then return the actual delay to use for that attempt.
//
// Example implementations:
//   - Fixed delay: always return baseDelay
//   - Linear backoff: return baseDelay * attempt
//   - Exponential backoff: return baseDelay * 2^attempt
type DelayTypeFunc func(attempt int, baseDelay, maxDelay time.Duration) time.Duration

// RetryConfig holds the complete configuration for retry behavior.
// It encapsulates all retry parameters including attempts, delays, logging,
// and delay calculation strategy. Use NewRetry() to create instances with
// sensible defaults and functional options for customization.
type RetryConfig struct {
	attempts  int           // Number of retry attempts
	baseDelay time.Duration // Base delay between attempts
	maxDelay  time.Duration // Maximum delay cap
	delayType DelayTypeFunc // Delay calculation strategy
	logger    Logger        // Logger for retry events
}

// NewRetry creates a new RetryConfig with sensible default values and applies
// the provided functional options. The defaults are designed for common use cases
// but can be easily customized using the With* option functions.
//
// Default configuration:
//   - 3 retry attempts
//   - 100ms base delay
//   - 1s maximum delay
//   - Fixed delay strategy
//   - Silent logging (nopLogger)
//
// Example:
//
//	config := retry.NewRetry(
//	    retry.WithAttempts(5),
//	    retry.WithDelay(200*time.Millisecond),
//	    retry.WithDelayType(retry.ExpBackoffWithJitter()),
//	)
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

// RetryFunc defines the signature for operations that can be retried.
// Currently specialized for operations returning io.ReadCloser (like HTTP responses).
// The function should return the resource and any error that occurred.
//
// Note: Future versions may support generic return types.
//
// Example:
//
//	retryFunc := func() (io.ReadCloser, error) {
//	    resp, err := http.Get("https://api.example.com/data")
//	    if err != nil {
//	        return nil, err
//	    }
//	    return resp.Body, nil
//	}
type RetryFunc func() (io.ReadCloser, error)

// Do executes the retry logic with the provided context and retry function.
// It attempts the operation up to the configured number of times, with delays
// between attempts calculated by the configured delay strategy.
//
// The method handles:
//   - Context cancellation (respects ctx.Done())
//   - Non-retryable errors (marked with NonRetryable())
//   - Delay calculation and sleeping between attempts
//   - Comprehensive logging of retry events
//
// Returns the successful result or the last error encountered after all
// attempts have been exhausted.
//
// Example:
//
//	ctx := context.WithTimeout(context.Background(), 30*time.Second)
//	result, err := config.Do(ctx, retryFunc)
//	if err != nil {
//	    log.Fatal("All retry attempts failed:", err)
//	}
//	defer result.Close()
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
