package slogger

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"
)

type (
	FormatMode string
	SourceMode string
)

const (
	FormatModeText FormatMode = "text"
	FormatModeJSON FormatMode = "json"

	SourceModeNone  SourceMode = "none"
	SourceModeShort SourceMode = "short"
	SourceModeFile  SourceMode = "file"
	SourceModeFull  SourceMode = "full"

	LevelTrace = slog.Level(-8)
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
	LevelFatal = slog.Level(12)
	LevelPanic = slog.Level(255)
)

type (
	Logger struct {
		*slog.Logger
	}

	logLevel struct {
		text string
	}
)

var (
	levelNames = map[slog.Leveler]logLevel{
		LevelTrace: {text: "TRACE"},
		LevelDebug: {text: "DEBUG"},
		LevelInfo:  {text: "INFO"},
		LevelWarn:  {text: "WARN"},
		LevelError: {text: "ERROR"},
		LevelFatal: {text: "FATAL"},
		LevelPanic: {text: "PANIC"},
	}
)

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

func (l *Logger) Slog() *slog.Logger {
	return l.Logger
}
