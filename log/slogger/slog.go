package slogger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	LevelTrace = slog.Level(-8)
	LevelFatal = slog.Level(12)
	LevelPanic = slog.Level(255)
)

type (
	Logger struct {
		*slog.Logger
	}

	HandlerOptions struct {
		*slog.HandlerOptions
		ShowTime   bool
		SourceMode SourceMode
	}

	SourceMode string
)

const (
	SourceModeFile  SourceMode = "file"
	SourceModeShort SourceMode = "short"
	SourceModeFull  SourceMode = "full"
)

var (
	levelNames = map[slog.Leveler]string{
		LevelTrace: "TRACE",
		LevelFatal: "FATAL",
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
		case slog.SourceKey:
			switch handler.SourceMode {
			case SourceModeFile:
				if src, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", src.File, src.Line))
				}
			case SourceModeShort:
				if src, ok := a.Value.Any().(*slog.Source); ok {
					fullPath := src.File
					seps := strings.Split(fullPath, "/")
					shortPath := seps[len(seps)-1]
					a.Value = slog.StringValue(fmt.Sprintf("%s:%d", shortPath, src.Line))
				}
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
	if !l.Enabled(context.Background(), LevelTrace) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Trace]
	r := slog.NewRecord(time.Now(), LevelTrace, msg, pcs[0])
	_ = l.Handler().Handle(context.Background(), r)
}

func (l *Logger) Fatal(msg string, fields ...any) {
	if !l.Enabled(context.Background(), LevelFatal) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Fatal]
	r := slog.NewRecord(time.Now(), LevelFatal, msg, pcs[0])
	_ = l.Handler().Handle(context.Background(), r)

	os.Exit(1)
}

func (l *Logger) Panic(msg string, fields ...any) {
	if !l.Enabled(context.Background(), LevelPanic) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(2, pcs[:]) // skip [Callers, Panic]
	r := slog.NewRecord(time.Now(), LevelPanic, msg, pcs[0])
	_ = l.Handler().Handle(context.Background(), r)
	panic(msg)
}

func (l *Logger) With(args ...any) *Logger {
	return &Logger{l.Logger.With(args...)}
}

func (l *Logger) WithGroup(name string) *Logger {
	return &Logger{l.Logger.WithGroup(name)}
}
