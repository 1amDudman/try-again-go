package retry

import (
	"log"
	"testing"
	"time"
)

// TestWithAttempts verifies that WithAttempts option correctly sets
// the number of retry attempts in RetryConfig.
func TestWithAttempts(t *testing.T) {
	r := NewRetry(WithAttempts(5))

	if r.attempts != 5 {
		t.Errorf("expected attempts to be 5, got %d", r.attempts)
	}
}

// TestWithDelay verifies that WithDelay option correctly sets
// the base delay duration in RetryConfig.
func TestWithDelay(t *testing.T) {
	r := NewRetry(WithDelay(11 * time.Second))

	if r.baseDelay != 11*time.Second {
		t.Errorf("expected baseDelay to be 11 seconds, got %v", r.baseDelay)
	}
}

// TestWithMaxDelay verifies that WithMaxDelay option correctly sets
// the maximum delay duration in RetryConfig.
func TestWithMaxDelay(t *testing.T) {
	r := NewRetry(WithMaxDelay(20 * time.Second))

	if r.maxDelay != 20*time.Second {
		t.Errorf("expected maxDelay to be 20 seconds, got %v", r.maxDelay)
	}
}

// TestWithDelayType verifies that WithDelayType option correctly sets
// the delay type function in RetryConfig.
func TestWithDelayType(t *testing.T) {
	customDelayFunc := func(int, time.Duration, time.Duration) time.Duration { return 123 }
	r := NewRetry(WithDelayType(customDelayFunc))

	if r.delayType(0, 0, 0) != 123 {
		t.Errorf("expected delayType to return 123")
	}
}

// TestWithLogger verifies that WithLogger option correctly sets
// the logger instance in RetryConfig.
func TestWithLogger(t *testing.T) {
	logger := log.Default()
	r := NewRetry(WithLogger(logger))

	if r.logger != logger {
		t.Errorf("expected logger to be set, got %v", r.logger)
	}
}

// TestFixedDelay verifies that FixedDelay function returns a DelayTypeFunc
// that always returns the base delay regardless of attempt number.
func TestFixedDelay(t *testing.T) {
	attempt := 5
	baseDelay := 10 * time.Second
	maxDelay := 3 * time.Second
	delayFunc := FixedDelay()

	delay := delayFunc(attempt, baseDelay, maxDelay)
	if delay != baseDelay {
		t.Errorf("expected fixed delay to return baseDelay, got %v", delay)
	}
}

// TestExpBackoffWithJitterUpperBound verifies that ExpBackoffWithJitter
// respects the maximum delay limit and never exceeds it, even with jitter.
// This test runs multiple iterations to account for random jitter values.
func TestExpBackoffWithJitterUpperBound(t *testing.T) {
	attempt := 2
	baseDelay := 100 * time.Millisecond
	maxDelay := 300 * time.Millisecond
	delayFunc := ExpBackoffWithJitter()

	for i := 0; i < 1000; i++ {
		delay := delayFunc(attempt, baseDelay, maxDelay)
		if delay > maxDelay {
			t.Errorf("delay exceeded upper bound: got %v, want <= %v", delay, maxDelay)
		}
	}
}

// TestExpBackoffWithJitterLowerBound verifies that ExpBackoffWithJitter
// returns delays that are at least equal to the exponential backoff base value.
// This test ensures that jitter only adds to the delay, never subtracts from it.
func TestExpBackoffWithJitterLowerBound(t *testing.T) {
	attempt := 3
	baseDelay := 100 * time.Millisecond
	maxDelay := 1000 * time.Millisecond
	delayFunc := ExpBackoffWithJitter()
	expBackoff := baseDelay * time.Duration(1<<(attempt-1))

	for i := 0; i < 1000; i++ {
		delay := delayFunc(attempt, baseDelay, maxDelay)
		if delay < expBackoff {
			t.Errorf("delay %v is less than expected minimum %v", delay, expBackoff)
		}
	}
}

// TestExpBackoffWithJitter_EdgeCases verifies that the delay calculation
// correctly handles boundary conditions, such as the initial attempt floor (0)
// and the maximum delay cap.
func TestExpBackoffWithJitter_EdgeCases(t *testing.T) {
	t.Parallel()
	delayFunc := ExpBackoffWithJitter()

	testCases := []struct {
		name        string
		attempt     int
		baseDelay   time.Duration
		maxDelay    time.Duration
		minExpected time.Duration
		maxExpected time.Duration
	}{
		{
			name:        "bitwise shift floor (attempt 0)",
			attempt:     0,
			baseDelay:   100 * time.Millisecond,
			maxDelay:    1000 * time.Millisecond,
			minExpected: 100 * time.Millisecond,
			maxExpected: 120 * time.Millisecond,
		},
		{
			name:        "finalDelay > maxDelay cap",
			attempt:     5,
			baseDelay:   100 * time.Millisecond,
			maxDelay:    200 * time.Millisecond,
			minExpected: 200 * time.Millisecond,
			maxExpected: 200 * time.Millisecond,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			delay := delayFunc(tc.attempt, tc.baseDelay, tc.maxDelay)

			if delay < tc.minExpected || delay > tc.maxExpected {
				t.Errorf("expected delay between %v and %v, got %v",
					tc.minExpected, tc.maxExpected, delay)
			}
		})
	}
}
