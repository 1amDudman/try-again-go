package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// TestDoSuccessCases tests the Do method for scenarios where the retry function
// eventually succeeds. It covers cases where the first attempt succeeds and
// where a retry is needed before success.
func TestDoSuccessCases(t *testing.T) {
	testCases := []struct {
		name          string
		retryFunc     func(calls *int) (string, error)
		expectedCalls int
		expectedData  string
	}{
		{
			name: "First Try Success",
			retryFunc: func(calls *int) (string, error) {
				*calls++
				return "success", nil
			},
			expectedCalls: 1,
			expectedData:  "success",
		},
		{
			name: "Success After First Try Failure",
			retryFunc: func(calls *int) (string, error) {
				*calls++
				if *calls == 1 {
					return "", fmt.Errorf("first attempt error")
				}
				return "success", nil
			},
			expectedCalls: 2,
			expectedData:  "success",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rc := NewRetry()
			ctx := context.Background()
			calls := 0

			var firstAttemptErr error
			fn := func() (string, error) {
				result, err := tc.retryFunc(&calls)
				if calls == 1 {
					firstAttemptErr = err
				}
				return result, err
			}

			result, err := Do(ctx, rc, fn)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			if tc.expectedCalls > 1 && firstAttemptErr == nil {
				t.Error("expected first attempt to fail, but it didn't")
			}

			if calls != tc.expectedCalls {
				t.Fatalf("expected %d call(s), got %d", tc.expectedCalls, calls)
			}

			if result != tc.expectedData {
				t.Fatalf("expected '%s', got '%s'", tc.expectedData, result)
			}
		})
	}
}

// TestDoAllAttemptsFailed tests the Do method for scenarios where all retry
// attempts fail. It verifies that an error is returned and the result is empty
// when all attempts are unsuccessful.
func TestDoAllAttemptsFailed(t *testing.T) {
	rc := NewRetry()
	ctx := context.Background()

	fn := func() (string, error) {
		return "", fmt.Errorf("attempt error")
	}

	result, err := Do(ctx, rc, fn)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if result != "" {
		t.Fatalf("expected nothing, got '%s'", result)
	}
}

// TestDoNonRetryableError tests the Do method for scenarios where a
// non-retryable error is returned. It checks that the error is correctly
// identified as non-retryable and the result is empty.
func TestDoNonRetryableError(t *testing.T) {
	rc := NewRetry()
	ctx := context.Background()

	fn := func() (string, error) {
		return "", NonRetryable(fmt.Errorf("critical error"))
	}

	result, err := Do(ctx, rc, fn)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, errNonRetryable) {
		t.Fatalf("expected non-retryable error, got: %v", err)
	}

	if result != "" {
		t.Fatalf("expected nothing, got '%s'", result)
	}
}

// TestDoContextCancelation tests the Do method for scenarios where a
// context was canceled instantly or during delay. It checks that the error
// is thrown, the error is correct, and the number of calls.
func TestDoContextCancelation(t *testing.T) {
	tests := []struct {
		name          string
		configOpts    []Option
		setupCtx      func() (context.Context, context.CancelFunc)
		getRetryFunc  func(calls *int, cancel context.CancelFunc) RetryFunc[string]
		expectedCalls int
	}{
		{
			name:       "Instant cancel",
			configOpts: nil,
			setupCtx: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, cancel
			},
			getRetryFunc: func(calls *int, cancel context.CancelFunc) RetryFunc[string] {
				return func() (string, error) {
					*calls++
					return "", fmt.Errorf("attempt error")
				}
			},
			expectedCalls: 0,
		},
		{
			name:       "Cancel during delay",
			configOpts: []Option{WithDelay(2 * time.Second)},
			setupCtx: func() (context.Context, context.CancelFunc) {
				return context.WithCancel(context.Background())
			},
			getRetryFunc: func(calls *int, cancel context.CancelFunc) RetryFunc[string] {
				return func() (string, error) {
					*calls++
					if *calls == 1 {
						go func() {
							time.Sleep(100 * time.Millisecond)
							cancel()
						}()

						return "", fmt.Errorf("first attempt fail")
					}

					return "", nil
				}
			},
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := NewRetry(tt.configOpts...)
			ctx, cancel := tt.setupCtx()
			defer cancel()

			calls := 0
			fn := tt.getRetryFunc(&calls, cancel)

			result, err := Do(ctx, rc, fn)
			if err == nil {
				t.Fatalf("expected error due to context cancelation, got nil")
			}

			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context canceled error, got %v", err)
			}

			if calls != tt.expectedCalls {
				t.Errorf("expected %d calls, got %d", tt.expectedCalls, calls)
			}

			if result != "" {
				t.Fatalf("expected nothing, got '%s'", result)
			}
		})
	}
}
