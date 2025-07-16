package retry

import (
	"errors"
	"fmt"
	"net"
)

// errNonRetryable is a sentinel error for non-retryable operations.
var errNonRetryable = errors.New("non-retryable error")

// NonRetryable wraps an error to indicate it is not retryable.
func NonRetryable(err error) error {
	return fmt.Errorf("%w: %v", errNonRetryable, err)
}

// isRetryable checks if an error is retryable.
func isRetryable(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return !errors.Is(err, errNonRetryable)
}
