package shared

import (
	"context"

	"github.com/charmbracelet/log"

	"github.com/peterldowns/pgmigrate"
)

type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

type LogAdapter struct {
	*log.Logger
}

func (l LogAdapter) Log(_ context.Context, level pgmigrate.LogLevel, msg string, fields ...pgmigrate.LogField) {
	args := make([]any, 0, 2*len(fields))
	for _, field := range fields {
		args = append(args, field.Key, field.Value)
	}
	switch level {
	case pgmigrate.LogLevelDebug:
		l.Logger.Debug(msg, args...)
	case pgmigrate.LogLevelInfo:
		l.Logger.Info(msg, args...)
	case pgmigrate.LogLevelError:
		l.Logger.Error(msg, args...)
	case pgmigrate.LogLevelWarning:
		l.Logger.Warn(msg, args...)
	}
}
