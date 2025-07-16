package retry

import (
	"errors"
	"testing"
)

func TestIsRetryable_DefaultError(t *testing.T) {
	err := errors.New("some error")
	if !isRetryable(err) {
		t.Error("default error should be retryable")
	}
}

func TestIsRetryable_NonRetryableError(t *testing.T) {
	err := NonRetryable(errors.New("fatal"))
	if isRetryable(err) {
		t.Error("non-retryable error should not be retryable")
	}
}

type timeoutError struct{}

func (timeoutError) Error() string   { return "timeout error" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return false }

func TestIsRetryable_TimeoutError(t *testing.T) {
	err := timeoutError{}
	if !isRetryable(err) {
		t.Error("timeout error should be retryable")
	}
}
