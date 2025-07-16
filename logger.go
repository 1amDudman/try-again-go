package retry

// Logger interface defines the logging behavior for retries.
type Logger interface {
	Printf(format string, v ...any)
}

// nopLogger is a no-operation logger that does nothing.
type nopLogger struct{}

// Printf implements the Logger interface for nopLogger.
func (nopLogger) Printf(string, ...any) {}
