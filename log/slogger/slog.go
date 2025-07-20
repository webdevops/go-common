package slogger

import (
	"context"
	"log/slog"
	"os"
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

	if handler.ReplaceAttr != nil {
		handler.ReplaceAttr = NewReplaceAttr(ret, handler.ReplaceAttr)
	} else {
		handler.ReplaceAttr = NewReplaceAttr(ret, func(groups []string, a slog.Attr) slog.Attr { return a })
	}

	return ret
}

func NewReplaceAttr(handler *HandlerOptions, callback func(groups []string, a slog.Attr) slog.Attr) func(groups []string, a slog.Attr) slog.Attr {
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

		return callback(groups, a)
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
	os.Exit(1)
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
