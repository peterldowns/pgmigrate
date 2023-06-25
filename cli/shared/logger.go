package shared

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/log"

	"github.com/peterldowns/pgmigrate"
)

type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

func NewLogger() (*log.Logger, LogAdapter) {
	var logger *log.Logger
	switch *Flags.LogFormat {
	case string(LogFormatText):
		logger = log.NewWithOptions(os.Stdout, log.Options{Formatter: log.TextFormatter})
	case string(LogFormatJSON):
		logger = log.NewWithOptions(os.Stdout, log.Options{Formatter: log.JSONFormatter})
	default:
		panic(fmt.Errorf("unknown log format: %s", *Flags.LogFormat))
	}
	return logger, LogAdapter{logger}
}

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
	}
}
