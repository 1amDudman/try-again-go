# try-again-go

A simple and flexible retry library for Go operations.

## Features

- 🔄 Configurable number of retry attempts
- 🧩 Type-safe Generics: Works with any type T effortlessly. No interface casting, no reflection
- ⏱️ Flexible delay strategies (fixed, exponential backoff with jitter)
- 🎯 Smart retryable error detection
- 🚫 Context cancellation support
- 📝 Customizable logging
- 📊 Observability: Lifecycle hooks for metrics and monitoring
- 🏗️ Simple and clean API

## Installation

```bash
go get github.com/1amDudman/try-again-go
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "net/http"
    "time"

    retry "github.com/1amDudman/try-again-go"
)

func main() {
    // Create retry config with default settings
    retryConfig := retry.NewRetry()

    // Function to retry
    retryFunc := func() (string, error) {
        resp, err := http.Get("https://example.com")
        if err != nil {
            return "", err
        }

        return "success", nil
    }

    // Execute with retries
    ctx := context.Background()
    result, err := retry.Do(ctx, retryConfig, retryFunc)
    if err != nil {
        fmt.Printf("All attempts failed: %v\n", err)
        return
    }

    fmt.Println("Success!")
}
```

## Configuration

> **Important**: The library uses Go Generics. Your retry function can return any type T using the signature func() (T, error).

### Basic Parameters

```go
retryConfig := retry.NewRetry(
    retry.WithAttempts(5),                                    // 5 attempts
    retry.WithDelay(200*time.Millisecond),                   // base delay 200ms
    retry.WithMaxDelay(5*time.Second),                       // max delay 5s
    retry.WithDelayType(retry.ExpBackoffWithJitter()),       // exponential backoff with jitter
    retry.WithLogger(customLogger),                          // custom logger
    retry.WithOnRetry(metricsHook),                          // metrics collection hook
)
```

### Observability & Metrics

If you need to track retry behavior without parsing
text logs (e.g., for Prometheus or Datadog), use the OnRetry hook.
It executes after a failed attempt, right before the system "sleeps".
This is perfect for monitoring the stability of external services,
such as third-party APIs or client data synchronizations.

```go
retryConfig := retry.NewRetry(
    retry.WithOnRetry(func(attempt int, err error, delay time.Duration) {
        // Clean metric collection without log parsing
        metrics.IncRetryCount(err.Error())
        metrics.AddSleepTime(delay.Seconds()) 
    }),
)
```

### Delay Strategies

#### Fixed Delay
```go
retry.WithDelayType(retry.FixedDelay())
```

#### Exponential Backoff with Jitter
```go
retry.WithDelayType(retry.ExpBackoffWithJitter())
```

### Logging

```go
type CustomLogger struct{}

func (cl CustomLogger) Printf(format string, v ...any) {
    log.Printf("[RETRY] "+format, v...)
}

retryConfig := retry.NewRetry(
    retry.WithLogger(CustomLogger{}),
)
```

## Error Handling

### Non-Retryable Errors

Some errors should not be retried. Use `NonRetryable`:

```go
func riskyOperation() (string, error) {
    if someCondition {
        return "", retry.NonRetryable(errors.New("critical error"))
    }
    // ...
}
```

### Automatic Detection

The library automatically considers retryable:
- Network timeouts
- All errors except those marked as `NonRetryable`

## Operation Cancellation

Use context to cancel operations:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := retry.Do(ctx, retryConfig, retryFunc)
```

## Default Settings

- **Attempts**: 3
- **Base delay**: 100ms
- **Max delay**: 1s
- **Delay strategy**: Fixed
- **Logger**: No output
- **OnRetry**: No-op (silent)

## License

MIT