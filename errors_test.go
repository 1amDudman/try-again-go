package retry

import (
	"errors"
	"testing"
)

// TestIsRetryableDefaultError verifies that regular errors are considered
// retryable by default. This ensures that common errors like network issues
// or temporary failures will trigger retry logic.
func TestIsRetryableDefaultError(t *testing.T) {
	err := errors.New("some error")
	if !isRetryable(err) {
		t.Error("default error should be retryable")
	}
}

// TestIsRetryableNonRetryableError verifies that errors wrapped with
// NonRetryable() are correctly identified as non-retryable. This prevents
// infinite retry loops for critical errors that should fail immediately.
func TestIsRetryableNonRetryableError(t *testing.T) {
	err := NonRetryable(errors.New("fatal"))
	if isRetryable(err) {
		t.Error("non-retryable error should not be retryable")
	}
}

// timeoutError is a mock implementation of net.Error interface
// used for testing timeout error detection in retry logic.
type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout error" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return false }

// TestIsRetryableTimeoutError verifies that network timeout errors
// are correctly identified as retryable. This is crucial for handling
// transient network issues that can be resolved with retry attempts.
func TestIsRetryableTimeoutError(t *testing.T) {
	err := timeoutError{}
	if !isRetryable(err) {
		t.Error("timeout error should be retryable")
	}
}
