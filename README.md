# try-again-go

A simple and flexible retry library for Go operations.

> **Note**: Currently, this library is designed specifically for operations that return `io.ReadCloser` which comes from resp.Body. Support for generic return types is planned for future versions.

## Features

- 🔄 Configurable number of retry attempts
- ⏱️ Flexible delay strategies (fixed, exponential backoff with jitter)
- 🎯 Smart retryable error detection
- 🚫 Context cancellation support
- 📝 Customizable logging
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
    "io"
    "net/http"
    "time"

    "github.com/1amDudman/try-again-go"
)

func main() {
    // Create retry config with default settings
    retryConfig := retry.NewRetry()

    // Function to retry - must return (io.ReadCloser, error)
    retryFunc := func() (io.ReadCloser, error) {
        resp, err := http.Get("https://example.com")
        if err != nil {
            return nil, err
        }
        return resp.Body, nil
    }

    // Execute with retries
    ctx := context.Background()
    result, err := retryConfig.Do(ctx, retryFunc)
    if err != nil {
        fmt.Printf("All attempts failed: %v\n", err)
        return
    }
    defer result.Close()

    fmt.Println("Success!")
}
```

## Configuration

> **Important**: All retry functions must have the signature `func() (io.ReadCloser, error)`. This is the current implementation limitation.

### Basic Parameters

```go
retryConfig := retry.NewRetry(
    retry.WithAttempts(5),                                    // 5 attempts
    retry.WithDelay(200*time.Millisecond),                   // base delay 200ms
    retry.WithMaxDelay(5*time.Second),                       // max delay 5s
    retry.WithDelayType(retry.ExpBackoffWithJitter()),       // exponential backoff with jitter
    retry.WithLogger(customLogger),                          // custom logger
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
func riskyOperation() (io.ReadCloser, error) {
    if someCondition {
        return nil, retry.NonRetryable(errors.New("critical error"))
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

result, err := retryConfig.Do(ctx, retryFunc)
```

## Default Settings

- **Attempts**: 3
- **Base delay**: 100ms
- **Max delay**: 1s
- **Delay strategy**: Fixed
- **Logger**: No output

## License

MIT