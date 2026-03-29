package retry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// TestDoSuccessCases tests the Do method for scenarios where the retry function
// eventually succeeds. It covers cases where the first attempt succeeds and
// where a retry is needed before success.
func TestDoSuccessCases(t *testing.T) {
	testCases := []struct {
		name          string
		retryFunc     func(calls *int) (io.ReadCloser, error)
		expectedCalls int
		expectedData  string
	}{
		{
			name: "First Try Success",
			retryFunc: func(calls *int) (io.ReadCloser, error) {
				*calls++
				return io.NopCloser(strings.NewReader("success")), nil
			},
			expectedCalls: 1,
			expectedData:  "success",
		},
		{
			name: "Success After First Try Failure",
			retryFunc: func(calls *int) (io.ReadCloser, error) {
				*calls++
				if *calls == 1 {
					return nil, fmt.Errorf("first attempt error")
				}
				return io.NopCloser(strings.NewReader("success")), nil
			},
			expectedCalls: 2,
			expectedData:  "success",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewRetry()
			ctx := context.Background()
			calls := 0

			var firstAttemptErr error
			wrappedRetryFunc := func() (io.ReadCloser, error) {
				result, err := tc.retryFunc(&calls)
				if calls == 1 {
					firstAttemptErr = err
				}
				return result, err
			}

			result, err := r.Do(ctx, wrappedRetryFunc)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			defer func() {
				if err := result.Close(); err != nil {
					t.Fatalf("error closing result: %v", err)
				}
			}()

			if tc.expectedCalls > 1 && firstAttemptErr == nil {
				t.Error("expected first attempt to fail, but it didn't")
			}

			if calls != tc.expectedCalls {
				t.Fatalf("expected %d call(s), got %d", tc.expectedCalls, calls)
			}

			data, err := io.ReadAll(result)
			if err != nil {
				t.Fatalf("error reading result: %v", err)
			}

			if string(data) != tc.expectedData {
				t.Fatalf("expected '%s', got '%s'", tc.expectedData, data)
			}
		})
	}
}

// TestDoAllAttemptsFailed tests the Do method for scenarios where all retry
// attempts fail. It verifies that an error is returned and the result is nil
// when all attempts are unsuccessful.
func TestDoAllAttemptsFailed(t *testing.T) {
	r := NewRetry()
	ctx := context.Background()

	retryFunc := func() (io.ReadCloser, error) {
		return nil, fmt.Errorf("attempt error")
	}

	result, err := r.Do(ctx, retryFunc)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	assertNilResult(t, result)
}

// TestDoNonRetryableError tests the Do method for scenarios where a
// non-retryable error is returned. It checks that the error is correctly
// identified as non-retryable and the result is nil.
func TestDoNonRetryableError(t *testing.T) {
	r := NewRetry()
	ctx := context.Background()

	retryFunc := func() (io.ReadCloser, error) {
		return nil, NonRetryable(fmt.Errorf("critical error"))
	}

	result, err := r.Do(ctx, retryFunc)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if !errors.Is(err, errNonRetryable) {
		t.Fatalf("expected non-retryable error, got: %v", err)
	}

	assertNilResult(t, result)
}

// TestDoContextCancelation tests the Do method for scenarios where a
// context was canceled instantly or during delay. It checks that the error
// is thrown, the error is correct, and the number of calls.
func TestDoContextCancelation(t *testing.T) {
	tests := []struct {
		name          string
		configOpts    []Option
		setupCtx      func() (context.Context, context.CancelFunc)
		getRetryFunc  func(calls *int, cancel context.CancelFunc) RetryFunc
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
			getRetryFunc: func(calls *int, cancel context.CancelFunc) RetryFunc {
				return func() (io.ReadCloser, error) {
					*calls++
					return nil, fmt.Errorf("attempt error")
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
			getRetryFunc: func(calls *int, cancel context.CancelFunc) RetryFunc {
				return func() (io.ReadCloser, error) {
					*calls++
					if *calls == 1 {
						go func() {
							time.Sleep(100 * time.Millisecond)
							cancel()
						}()

						return nil, fmt.Errorf("first attempt fail")
					}

					return nil, nil
				}
			},
			expectedCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRetry(tt.configOpts...)
			ctx, cancel := tt.setupCtx()
			defer cancel()

			calls := 0
			retryFunc := tt.getRetryFunc(&calls, cancel)

			result, err := r.Do(ctx, retryFunc)
			if err == nil {
				t.Fatalf("expected error due to context cancelation, got nil")
			}

			if !errors.Is(err, context.Canceled) {
				t.Fatalf("expected context canceled error, got %v", err)
			}

			if calls != tt.expectedCalls {
				t.Errorf("expected %d calls, got %d", tt.expectedCalls, calls)
			}

			assertNilResult(t, result)
		})
	}
}

// mockReadCloser structure is a dependency
// for TestDoCloseDataOnError fuction.
type mockReadCloser struct {
	closed bool
}

func (m *mockReadCloser) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func (m *mockReadCloser) Close() error {
	m.closed = true
	return nil
}

// TestDoCloseDataOnError tests the Do method for unexpected
// resource leaks when an error is returned with data,
// and data wasn't closed properly.
func TestDoCloseDataOnError(t *testing.T) {
	r := NewRetry(WithAttempts(2))
	ctx := context.Background()

	mockData := &mockReadCloser{}

	retryFunc := func() (io.ReadCloser, error) {
		return mockData, fmt.Errorf("some tmp error")
	}

	result, err := r.Do(ctx, retryFunc)
	if err == nil {
		t.Fatalf("expected error after all attempts failed, got nil")
	}

	if !mockData.closed {
		t.Error("expected data.Close() to be called, but it was not")
	}

	assertNilResult(t, result)
}

// assertNilResult is a helper function to assert that the result is nil and
// handle closing if not. It fails the test if the result is not nil.
func assertNilResult(t *testing.T, result io.ReadCloser) {
	t.Helper()
	if result != nil {
		defer result.Close()
		t.Fatalf("expected result to be nil on error, got non-nil")
	}
}
