package logging

import (
	"context"
	"fmt"
	"testing"
)

// NewTestLogger returns a [TestLogger], which is a [migrate.TestLoggerWithHelper] that
// writes all logs to a given test's output in such a way that stack traces are
// correctly preserved.
func NewTestLogger(t *testing.T) TestLogger {
	return TestLogger{t}
}

// TestLogger implements the migrate.TestLoggerWithHelper interface and writes all logs
// to a given test's output in such a way that stack traces are correctly
// preserved.
type TestLogger struct {
	*testing.T
}

// Log writes a message to a given test's output in pseudo key=value form.
func (t TestLogger) Log(_ context.Context, level Level, msg string, fields ...Field) {
	t.Helper()
	line := fmt.Sprintf("%s: %s", level, msg)
	for _, field := range fields {
		line = fmt.Sprintf("%s %s=%v", line, field.Key, field.Value)
	}
	t.T.Log(line)
}
