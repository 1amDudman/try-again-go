package retry

import (
	"context"
	"fmt"
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

// OnRetryFunc defines a signature for a lifecycle hook
// executed after a failed attempt, right before the delay.
type OnRetryFunc func(attempt int, err error, delay time.Duration)

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
	onRetry   OnRetryFunc   // TODO
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
		onRetry:   func(attempt int, err error, delay time.Duration) {},
	}

	for _, opt := range opts {
		opt(retry)
	}

	// maxDelay validation in case a client forgot to set maxDelay
	// with baseDelay or set it less than baseDelay
	if retry.maxDelay < retry.baseDelay {
		retry.maxDelay = retry.baseDelay
	}

	return retry
}

// RetryFunc defines the signature for operations that can be retried.
// The function should return the resource and any error that occurred.
//
//
// Example:

// Be careful, the "string" is for reference here, you have to
// replace with your specific type.
//	retryFunc := func() (string, error) {
//	    resp, err := http.Get("https://api.example.com/data")
//	    if err != nil {
//	        return "", err
//	    }
//		defer resp.Body.Close()

//	    return "success", nil
//	}
type RetryFunc[T any] func() (T, error)

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
//	result, err := Do(ctx, config, retryFunc)
//	if err != nil {
//	    log.Fatal("All retry attempts failed:", err)
//	}
func Do[T any](ctx context.Context, rc *RetryConfig, fn RetryFunc[T]) (T, error) {
	var zero T
	var lastErr error

	for attempt := 1; attempt <= rc.attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			rc.logger.Printf("Context canceled before attempt %d: %v", attempt, err)
			return zero, fmt.Errorf("context canceled before attempt %d: %w", attempt, err)
		}

		data, err := fn()
		if err == nil {
			return data, nil
		}

		lastErr = err

		if !isRetryable(err) {
			rc.logger.Printf("Non-retryable error on attempt %d: %v", attempt, err)
			return zero, fmt.Errorf("non-retryable error: %w", err)
		}

		if attempt == rc.attempts {
			break
		}

		delay := rc.baseDelay
		if rc.delayType != nil {
			delay = rc.delayType(attempt, rc.baseDelay, rc.maxDelay)
		}

		rc.onRetry(attempt, err, delay)

		rc.logger.Printf("Attempt %d failed: %v. Retrying in %v...\n", attempt, err, delay)

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			rc.logger.Printf("Retry canceled by context on attempt %d: %v", attempt, ctx.Err())
			return zero, fmt.Errorf("retry canceled by context on attempt %d: %w", attempt, ctx.Err())
		case <-timer.C:
		}
	}

	rc.logger.Printf("All %d attempts failed. Last error: %v", rc.attempts, lastErr)
	return zero, fmt.Errorf("all attempts failed, the last error: %w", lastErr)
}
