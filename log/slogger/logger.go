package slogger

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
)

type LoggerOptionFunc func(*Options)

// WithLevelText sets the log level as string
func WithLevelText(val string) LoggerOptionFunc {
	level, err := TranslateToLogLevel(val)
	if err != nil {
		panic(err)
	}

	return func(opt *Options) {
		opt.Level = level
	}
}

// WithLevel sets the log level as slog.Level
func WithLevel(level slog.Level) LoggerOptionFunc {
	return func(opt *Options) {
		opt.Level = level
	}
}

// WithSourceMode sets the source (file, line) mode
func WithSourceMode(mode SourceMode) LoggerOptionFunc {
	switch mode {
	case "":
		mode = SourceModeNone
	case SourceModeNone:
	case SourceModeFile:
	case SourceModeShort:
	case SourceModeFull:
	default:
		panic(fmt.Errorf(`unknown source mode "%s"`, mode))
	}

	return func(opt *Options) {
		opt.SourceMode = mode
		if mode != SourceModeNone {
			opt.AddSource = true
		}
	}
}

// WithFormat sets the mode (logfmt or json)
func WithFormat(mode FormatMode) LoggerOptionFunc {
	switch mode {
	case "":
		mode = FormatModeLogfmt
	case FormatModeLogfmt:
	case FormatModeJSON:
	default:
		panic(fmt.Errorf(`unknown format mode "%s"`, mode))
	}

	return func(opt *Options) {
		opt.Format = mode
	}
}

// WithColor sets if logs should be colorful or not
func WithColor(mode ColorMode) LoggerOptionFunc {
	return func(opt *Options) {
		opt.SetColorMode(mode)
	}
}

// WithTime sets if logs lines should also include the time (not useful for containers)
func WithTime(v bool) LoggerOptionFunc {
	return func(opt *Options) {
		opt.ShowTime = v
	}
}

// NewFromSlog converts a slog.Logger to an slogger.Logger
func NewFromSlog(logger *slog.Logger) *Logger {
	return &Logger{logger}
}

// NewDiscardLogger creates a logger which discards all logs (send logs to /dev/null)
func NewDiscardLogger() *Logger {
	loggerOptions := NewOptions(nil)
	return newLoggerHandler(io.Discard, loggerOptions)
}

// NewDaemonLogger creates a logger with defaults for daemons
func NewDaemonLogger(w io.Writer, opts ...LoggerOptionFunc) *Logger {
	loggerOptions := NewOptions(nil).
		SetLevel(LevelInfo).
		SetSourceMode(SourceModeFile).
		SetFormat(FormatModeLogfmt)

	return newLoggerHandler(w, loggerOptions, opts...)
}

// NewCliLogger creates a cli logger with defaults for cli tasks
func NewCliLogger(w io.Writer, opts ...LoggerOptionFunc) *Logger {
	loggerOptions := NewOptions(nil).
		SetLevel(LevelInfo).
		SetSourceMode(SourceModeNone).
		SetFormat(FormatModeLogfmt).
		SetShowTime(false).
		SetColorMode(ColorModeAuto)

	return newLoggerHandler(w, loggerOptions, opts...)
}

func newLoggerHandler(w io.Writer, handlerOpts *Options, opts ...LoggerOptionFunc) *Logger {
	var handler slog.Handler
	for _, opt := range opts {
		opt(handlerOpts)
	}

	switch handlerOpts.Format {
	case FormatModeJSON:
		handler = slog.NewJSONHandler(w, handlerOpts.HandlerOptions)
	default:
		if handlerOpts.Color {
			tintOpts := tint.Options{
				ReplaceAttr: handlerOpts.ReplaceAttr,
				Level:       handlerOpts.Level,
				AddSource:   handlerOpts.SourceMode != SourceModeNone,
				NoColor:     false,
			}

			if !handlerOpts.ShowTime {
				tintOpts.TimeFormat = ""
			}

			handler = tint.NewHandler(w, &tintOpts)
		} else {
			handler = slog.NewTextHandler(w, handlerOpts.HandlerOptions)
		}
	}

	logger := New(handler)
	return logger
}
