package pgmigrate

import "context"

// LogLevel represents the severity of the log message, and is one of
//   - [LogLevelDebug]
//   - [LogLevelInfo]
//   - [LogLevelError]
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelError   LogLevel = "error"
	LogLevelWarning LogLevel = "warning"
)

// LogField holds a key/value pair for structured logging.
type LogField struct {
	Key   string
	Value any
}

// Logger is a generic logging interface so that you can easily use pgmigrate
// with your existing structured ogging solution -- hopefully it is not
// difficult for you to write an adapter.
type Logger interface {
	Log(context.Context, LogLevel, string, ...LogField)
}

// Helper is an optional interface that your logger can implement to help
// make debugging and stacktraces easier to understand, primarily in tests.
// If a [Logger] implements this interface, pgmigrate will call Helper()
// in its own helper methods for writing to your logger, with the goal of
// omitting pgmigrate's helper methods from your stacktraces.
//
// For instance, the [TestLogger] we provide embeds a [testing.T], which
// implements Helper().
//
// You do *not* need to implement this interface in order for pgmigrate
// to successfully use your logger.
type Helper interface {
	Helper()
}
