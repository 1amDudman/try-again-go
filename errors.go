package retry

import (
	"errors"
	"fmt"
	"net"
)

// errNonRetryable is a sentinel error used to mark operations that should not
// be retried. This prevents infinite retry loops for critical failures.
var errNonRetryable = errors.New("non-retryable error")

// NonRetryable wraps an error to explicitly mark it as non-retryable.
// Use this function to prevent retry attempts for critical errors like
// authentication failures, invalid input, or configuration errors.
//
// Example:
//
//	if unauthorized {
//	    return nil, retry.NonRetryable(errors.New("invalid credentials"))
//	}
func NonRetryable(err error) error {
	return fmt.Errorf("%w: %v", errNonRetryable, err)
}

// isRetryable determines whether an error should trigger a retry attempt.
// It returns true for network timeout errors and all errors except those
// explicitly marked as non-retryable using NonRetryable().
//
// The function follows this logic:
//   - Network timeout errors (net.Error with Timeout() == true) are retryable
//   - Errors wrapped with NonRetryable() are not retryable
//   - All other errors are retryable by default
func isRetryable(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return !errors.Is(err, errNonRetryable)
}
