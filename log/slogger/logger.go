package slogger

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/lmittmann/tint"
)

type LoggerOptionFunc func(*Options)

func WithLevelText(val string) LoggerOptionFunc {
	level, err := TranslateToLogLevel(val)
	if err != nil {
		panic(err)
	}

	return func(opt *Options) {
		opt.Level = level
	}
}

func WithLevel(level slog.Level) LoggerOptionFunc {
	return func(opt *Options) {
		opt.Level = level
	}
}

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

func WithColor(mode ColorMode) LoggerOptionFunc {
	return func(opt *Options) {
		opt.SetColorMode(mode)
	}
}

func WithTime(v bool) LoggerOptionFunc {
	return func(opt *Options) {
		opt.ShowTime = v
	}
}

func NewDaemonLogger(w io.Writer, opts ...LoggerOptionFunc) *Logger {
	loggerOptions := NewOptions(nil).
		SetLevel(LevelInfo).
		SetSourceMode(SourceModeFile).
		SetFormat(FormatModeLogfmt)

	return newLoggerHandler(w, loggerOptions, opts...)
}

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
