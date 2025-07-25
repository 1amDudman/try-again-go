package retry

// Logger interface defines the logging behavior for retry operations.
// Implementations should provide formatted logging output similar to fmt.Printf.
// This interface allows users to integrate their preferred logging solution
// (logrus, zap, standard log, etc.) with the retry library.
//
// Example usage:
//
//	type CustomLogger struct{}
//	func (cl CustomLogger) Printf(format string, v ...any) {
//	    log.Printf("[RETRY] "+format, v...)
//	}
//
//	retryConfig := retry.NewRetry(retry.WithLogger(CustomLogger{}))
type Logger interface {
	Printf(format string, v ...any)
}

// nopLogger is a no-operation logger implementation that discards all log output.
// It serves as the default logger when no custom logger is provided, ensuring
// silent operation without performance overhead from logging.
type nopLogger struct{}

// Printf implements the Logger interface for nopLogger by doing nothing.
// This allows the retry library to operate silently by default while still
// supporting the logging interface contract.
func (nopLogger) Printf(string, ...any) {}
