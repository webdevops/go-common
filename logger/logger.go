package logger

import (
	"context"
	"log/slog"
)

const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
	LevelPanic = slog.Level(255)
)

var (
	levelNames = map[slog.Leveler]string{
		LevelTrace: "TRACE",
		LevelFatal: "FATAL",
	}
)

type (
	Logger struct {
		*slog.Logger
	}

	HandlerOptions struct {
		*slog.HandlerOptions
		ShowTime bool
	}
)

func NewHandlerOptions(handler *slog.HandlerOptions) *HandlerOptions {
	if handler == nil {
		handler = &slog.HandlerOptions{}
	}

	ret := &HandlerOptions{HandlerOptions: handler}

	existingFunc := &handler.ReplaceAttr
	handler.ReplaceAttr = NewReplaceAttr(ret, existingFunc)

	return ret
}

func NewReplaceAttr(handler *HandlerOptions, existingFunc *func(groups []string, a slog.Attr) slog.Attr) func(groups []string, a slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		switch a.Key {
		case slog.LevelKey:
			level := a.Value.Any().(slog.Level)
			if levelLabel, exists := levelNames[level]; exists {
				a.Value = slog.StringValue(levelLabel)
			} else {
				a.Value = slog.StringValue(level.String())
			}
		case slog.TimeKey:
			if !handler.ShowTime {
				return slog.Attr{}
			}
		}

		if existingFunc != nil {
			return (*existingFunc)(groups, a)
		} else {
			return a
		}
	}
}

func New(handler slog.Handler) *Logger {
	return &Logger{
		Logger: slog.New(handler),
	}
}
func (l *Logger) Trace(msg string, fields ...any) {
	l.Log(context.Background(), LevelTrace, msg, fields...)
}

func (l *Logger) Fatal(msg string, fields ...any) {
	l.Log(context.Background(), LevelFatal, msg, fields...)
}

func (l *Logger) Panic(msg string, fields ...any) {
	l.Log(context.Background(), LevelPanic, msg, fields...)
	panic(msg)
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{l.Logger.With(args...)}
}

func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{l.Logger.WithGroup(name)}
}
