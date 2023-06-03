//nolint:gochecknoglobals
package main

import (
	"context"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib" // pgx driver
	"github.com/spf13/cobra"
	"golang.org/x/exp/slog"

	"github.com/peterldowns/pgmigrate/logging"
)

type LogFormat string

const (
	LogFormatJSON LogFormat = "json"
	LogFormatText LogFormat = "text"
)

var rootFlags struct {
	LogFormat *string
}

var root = &cobra.Command{
	Use:              "pgmigrate",
	Short:            "run migrations against a postgres database",
	TraverseChildren: true,
}

func main() {
	// Disable the builtin shell-completion script generator command
	root.CompletionOptions.DisableDefaultCmd = true
	rootFlags.LogFormat = root.PersistentFlags().String(
		"log-format",
		string(LogFormatText),
		fmt.Sprintf("'%s' (default) or '%s', the log line format", LogFormatText, LogFormatJSON),
	)
	if err := root.Execute(); err != nil {
		panic(err)
	}
}

func newLogger() (*slog.Logger, logging.Logger) {
	var logger *slog.Logger
	switch *rootFlags.LogFormat {
	case string(LogFormatText):
		logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	case string(LogFormatJSON):
		logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	default:
		panic(fmt.Errorf("unknown log format: %s", *rootFlags.LogFormat))
	}
	return logger, adaptedLogger{logger}
}

type adaptedLogger struct {
	*slog.Logger
}

func (l adaptedLogger) Log(ctx context.Context, level logging.Level, msg string, fields ...logging.Field) {
	slogLevel := map[logging.Level]slog.Level{
		logging.LevelDebug: slog.LevelDebug,
		logging.LevelInfo:  slog.LevelInfo,
		logging.LevelError: slog.LevelError,
	}[level]
	args := make([]any, 0, len(fields))
	for _, field := range fields {
		args = append(args, slog.Any(field.Key, field.Value))
	}
	l.Logger.Log(ctx, slogLevel, msg, args...)
}
