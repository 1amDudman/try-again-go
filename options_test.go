package retry

import (
	"log"
	"testing"
	"time"
)

func TestWithAttempts(t *testing.T) {
	r := NewRetry(WithAttempts(5))
	if r.attempts != 5 {
		t.Errorf("expected attempts to be 5, got %d", r.attempts)
	}
}

func TestWithDelay(t *testing.T) {
	r := NewRetry(WithDelay(11 * time.Second))
	if r.baseDelay != 11*time.Second {
		t.Errorf("expected baseDelay to be 11 seconds, got %v", r.baseDelay)
	}
}

func TestWithMaxDelay(t *testing.T) {
	r := NewRetry(WithMaxDelay(20 * time.Second))
	if r.maxDelay != 20*time.Second {
		t.Errorf("expected maxDelay to be 20 seconds, got %v", r.maxDelay)
	}
}

func TestWithDelayType(t *testing.T) {
	customDelayFunc := func(int, time.Duration, time.Duration) time.Duration { return 123 }
	r := NewRetry(WithDelayType(customDelayFunc))
	if r.delayType(0, 0, 0) != 123 {
		t.Errorf("expected delayType to return 123")
	}
}

func TestWithLogger(t *testing.T) {
	logger := log.Default()
	r := NewRetry(WithLogger(logger))
	if r.logger != logger {
		t.Errorf("expected logger to be set, got %v", r.logger)
	}
}

func TestFixedDelay(t *testing.T) {
	delay := FixedDelay()
	if delay(5, 10*time.Second, 3*time.Second) != 10*time.Second {
		t.Errorf("expected fixed delay to return baseDelay, got %v", delay(5, 10*time.Second, 3*time.Second))
	}
}

// TODO: incorrect test, should be updated
func TestExpBackoffWithJitter(t *testing.T) {
	delay := ExpBackoffWithJitter()
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second

	for i := 0; i < 5; i++ {
		d := delay(i, baseDelay, maxDelay)
		if d < baseDelay || d > maxDelay {
			t.Errorf("expected delay to be between %v and %v, got %v", baseDelay, maxDelay, d)
		}
	}
}
