package logging

import "context"

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelError Level = "error"
)

type Field struct {
	Key   string
	Value any
}

type Logger interface {
	Log(context.Context, Level, string, ...Field)
}

type Helper interface {
	Helper()
}
